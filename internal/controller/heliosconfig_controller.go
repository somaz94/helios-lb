/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	balancerv1 "github.com/somaz94/helios-lb/api/v1"
	"github.com/somaz94/helios-lb/internal/loadbalancer"
	"github.com/somaz94/helios-lb/internal/metrics"
	"github.com/somaz94/helios-lb/internal/network"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

// HeliosConfigReconciler reconciles a HeliosConfig object
type HeliosConfigReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	NetworkMgr *network.NetworkManager
	Balancer   *loadbalancer.LoadBalancer
	Metrics    *metrics.MetricsRecorder
}

// +kubebuilder:rbac:groups=balancer.helios.dev,resources=heliosconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=balancer.helios.dev,resources=heliosconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=balancer.helios.dev,resources=heliosconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HeliosConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
const (
	heliosConfigFinalizer = "balancer.helios.dev/finalizer"
)

func (r *HeliosConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	var heliosConfig balancerv1.HeliosConfig
	if err := r.Get(ctx, req.NamespacedName, &heliosConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 메트릭 기록
	defer func() {
		r.Metrics.RecordLBStatus(heliosConfig.Name, heliosConfig.Namespace,
			heliosConfig.Status.Phase == balancerv1.StateActive)
	}()

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(&heliosConfig, heliosConfigFinalizer) {
		controllerutil.AddFinalizer(&heliosConfig, heliosConfigFinalizer)
		if err := r.Update(ctx, &heliosConfig); err != nil {
			log.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the HeliosConfig is being deleted
	if !heliosConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &heliosConfig)
	}

	// List all services and filter LoadBalancer type
	var services corev1.ServiceList
	if err := r.List(ctx, &services); err != nil {
		return ctrl.Result{}, err
	}

	// Filter LoadBalancer services
	var lbServices []corev1.Service
	for _, svc := range services.Items {
		if svc.Spec.Type == corev1.ServiceTypeLoadBalancer {
			lbServices = append(lbServices, svc)
		}
	}
	services.Items = lbServices

	// Process each LoadBalancer service
	for _, svc := range services.Items {
		// heliosConfig에서 할당한 IP를 가진 서비스인지 확인
		isHeliosService := false
		if svc.Spec.LoadBalancerIP == heliosConfig.Spec.IPRange {
			isHeliosService = true
		}

		if len(svc.Status.LoadBalancer.Ingress) == 0 && isHeliosService {
			// IP 할당
			ip, err := r.NetworkMgr.AllocateIP(heliosConfig.Spec.IPRange)
			if err != nil {
				log.Error(err, "failed to allocate IP", "service", svc.Name)
				heliosConfig.Status.Phase = "Failed"
				heliosConfig.Status.Message = err.Error()
				if statusErr := r.Status().Update(ctx, &heliosConfig); statusErr != nil {
					log.Error(statusErr, "failed to update status")
				}
				return ctrl.Result{}, err
			}

			// 서비스 업데이트를 위한 재시도 로직
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				// 최신 서비스 상태 가져오기
				var currentSvc corev1.Service
				if err := r.Get(ctx, types.NamespacedName{
					Namespace: svc.Namespace,
					Name:      svc.Name,
				}, &currentSvc); err != nil {
					return err
				}

				// 어노테이션 추가
				if currentSvc.Annotations == nil {
					currentSvc.Annotations = make(map[string]string)
				}
				currentSvc.Annotations["balancer.helios.dev/load-balancer-class"] = "helios-lb"

				// spec에 loadBalancerClass 추가
				currentSvc.Spec.LoadBalancerClass = pointer.String("helios-lb")

				// 상태 업데이트도 함께
				currentSvc.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
					{IP: ip},
				}

				// 한 번에 업데이트
				if err := r.Status().Update(ctx, &currentSvc); err != nil {
					return err
				}

				return r.Update(ctx, &currentSvc)
			}); err != nil {
				log.Error(err, "failed to update service")
				continue
			}

			// HeliosConfig 상태 업데이트
			if heliosConfig.Status.AllocatedIPs == nil {
				heliosConfig.Status.AllocatedIPs = make(map[string]string)
			}
			heliosConfig.Status.AllocatedIPs[svc.Name] = ip
			heliosConfig.Status.Phase = balancerv1.StateActive
			heliosConfig.Status.Message = "IP allocated successfully"
			if err := r.Status().Update(ctx, &heliosConfig); err != nil {
				log.Error(err, "failed to update HeliosConfig status")
			}

			// Record metrics
			r.Metrics.RecordIPAllocation(ip, true)
		}
	}

	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// handleDeletion handles the deletion of a HeliosConfig
func (r *HeliosConfigReconciler) handleDeletion(ctx context.Context, heliosConfig *balancerv1.HeliosConfig) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(heliosConfig, heliosConfigFinalizer) {
		// Release all allocated IPs
		for serviceName, ip := range heliosConfig.Status.AllocatedIPs {
			r.NetworkMgr.ReleaseIP(ip)
			r.Metrics.RecordIPAllocation(ip, false)
			log.Info("released IP", "service", serviceName, "ip", ip)
		}

		// Remove finalizer
		controllerutil.RemoveFinalizer(heliosConfig, heliosConfigFinalizer)
		if err := r.Update(ctx, heliosConfig); err != nil {
			log.Error(err, "failed to remove finalizer")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HeliosConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&balancerv1.HeliosConfig{}).
		Watches(
			&corev1.Service{},
			handler.EnqueueRequestsFromMapFunc(r.findLoadBalancerServices),
			builder.WithPredicates(predicate.NewPredicateFuncs(func(obj client.Object) bool {
				svc, ok := obj.(*corev1.Service)
				if !ok {
					return false
				}
				return svc.Spec.Type == corev1.ServiceTypeLoadBalancer
			})),
		).
		Complete(r)
}

// findLoadBalancerServices watches for LoadBalancer type services
func (r *HeliosConfigReconciler) findLoadBalancerServices(ctx context.Context, obj client.Object) []reconcile.Request {
	// Type assertion check only
	if _, ok := obj.(*corev1.Service); !ok {
		return nil
	}

	// Get the HeliosConfig
	var heliosConfigs balancerv1.HeliosConfigList
	if err := r.List(ctx, &heliosConfigs); err != nil {
		return nil
	}

	// For now, we'll use the first HeliosConfig we find
	if len(heliosConfigs.Items) > 0 {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Name:      heliosConfigs.Items[0].Name,
					Namespace: heliosConfigs.Items[0].Namespace,
				},
			},
		}
	}
	return nil
}
