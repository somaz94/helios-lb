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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// HeliosConfigReconciler reconciles a HeliosConfig object
type HeliosConfigReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	NetworkMgr *network.NetworkManager
	Balancer   *loadbalancer.LoadBalancer
	Metrics    *metrics.MetricsRecorder
	IPMgr      *IPManager
	Recorder   record.EventRecorder
}

// +kubebuilder:rbac:groups=balancer.helios.dev,resources=heliosconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=balancer.helios.dev,resources=heliosconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=balancer.helios.dev,resources=heliosconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

const (
	heliosConfigFinalizer = "balancer.helios.dev/finalizer"
)

func (r *HeliosConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues(LogKeyConfig, req.Name, LogKeyNamespace, req.Namespace)
	reconcileStart := time.Now()

	var heliosConfig balancerv1.HeliosConfig
	if err := r.Get(ctx, req.NamespacedName, &heliosConfig); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger = logger.WithValues(LogKeyIPRange, heliosConfig.Spec.IPRange, LogKeyPhase, heliosConfig.Status.Phase)

	// Record metrics on defer
	defer func() {
		duration := time.Since(reconcileStart).Seconds()
		result := "success"
		if heliosConfig.Status.Phase == balancerv1.StateFailed {
			result = "error"
		}
		r.Metrics.RecordReconcileDuration(heliosConfig.Name, heliosConfig.Namespace, result, duration)
		r.Metrics.RecordLBStatus(heliosConfig.Name, heliosConfig.Namespace,
			heliosConfig.Status.Phase == balancerv1.StateActive)
		r.Metrics.RecordIPPoolUtilization(heliosConfig.Name, heliosConfig.Namespace,
			len(heliosConfig.Status.AllocatedIPs))
		logger.V(1).Info("reconcile complete",
			LogKeyReconcileTime, duration*1000,
			LogKeyAllocatedIPs, len(heliosConfig.Status.AllocatedIPs))
	}()

	// Add finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(&heliosConfig, heliosConfigFinalizer) {
		controllerutil.AddFinalizer(&heliosConfig, heliosConfigFinalizer)
		if err := r.Update(ctx, &heliosConfig); err != nil {
			logger.Error(err, "failed to add finalizer")
			return ctrl.Result{}, err
		}
		// Set initial condition
		meta.SetStatusCondition(&heliosConfig.Status.Conditions, metav1.Condition{
			Type:               balancerv1.ConditionTypeReady,
			Status:             metav1.ConditionFalse,
			Reason:             balancerv1.ReasonInitializing,
			Message:            "HeliosConfig is initializing",
			ObservedGeneration: heliosConfig.Generation,
		})
		if err := r.Status().Update(ctx, &heliosConfig); err != nil {
			logger.Error(err, "failed to set initial condition")
			return ctrl.Result{}, err
		}
	}

	// Check if the HeliosConfig is being deleted
	if !heliosConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &heliosConfig)
	}

	// List all services
	var serviceList corev1.ServiceList
	if err := r.List(ctx, &serviceList); err != nil {
		return ctrl.Result{}, err
	}

	// Filter eligible services using extracted logic
	eligible := FilterEligibleServices(
		serviceList.Items,
		heliosConfig.Spec.NamespaceSelector,
		heliosConfig.Spec.IPRange,
	)

	logger.V(1).Info("discovered eligible services", LogKeyServiceCount, len(eligible))

	// Check for IP conflicts with other HeliosConfigs
	conflicts, err := r.IPMgr.CheckIPConflicts(ctx, &heliosConfig)
	if err != nil {
		logger.Error(err, "failed to check IP conflicts")
		return ctrl.Result{}, err
	}
	if len(conflicts) > 0 {
		for ip, owner := range conflicts {
			logger.Info("IP conflict detected",
				LogKeyConflictIP, ip, LogKeyConflictOwner, owner)
		}
		r.Recorder.Eventf(&heliosConfig, corev1.EventTypeWarning, "IPConflict",
			"IP range overlaps with other HeliosConfigs: %d conflicting IP(s)", len(conflicts))
		meta.SetStatusCondition(&heliosConfig.Status.Conditions, metav1.Condition{
			Type:               balancerv1.ConditionTypeDegraded,
			Status:             metav1.ConditionTrue,
			Reason:             balancerv1.ReasonIPConflict,
			Message:            fmt.Sprintf("IP range conflicts with %d allocated IP(s) from other configs", len(conflicts)),
			ObservedGeneration: heliosConfig.Generation,
		})
		if statusErr := r.Status().Update(ctx, &heliosConfig); statusErr != nil {
			logger.Error(statusErr, "failed to update status after conflict detection")
		}
		r.Metrics.RecordRequeueReason(heliosConfig.Name, heliosConfig.Namespace, "ip_conflict")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Process each eligible service
	for i := range eligible {
		svc := &eligible[i]
		svcLogger := logger.WithValues(LogKeyService, svc.Name, LogKeyNamespace, svc.Namespace)

		// Check MaxAllocations quota
		if err := r.IPMgr.CheckQuota(&heliosConfig); err != nil {
			svcLogger.Info("max allocations reached, skipping remaining services",
				LogKeyMaxAlloc, heliosConfig.Spec.MaxAllocations,
				LogKeyCurrentAlloc, len(heliosConfig.Status.AllocatedIPs))
			r.Recorder.Eventf(&heliosConfig, corev1.EventTypeWarning, "QuotaExceeded",
				"Max allocations reached (%d/%d)", len(heliosConfig.Status.AllocatedIPs), heliosConfig.Spec.MaxAllocations)
			break
		}

		// Allocate IP and assign to service
		ip, err := r.IPMgr.AllocateAndAssign(ctx, svcLogger, &heliosConfig, svc)
		if err != nil {
			svcLogger.Error(err, "failed to allocate and assign IP")
			r.Recorder.Eventf(&heliosConfig, corev1.EventTypeWarning, "AllocationFailed",
				"Failed to allocate IP for service %s/%s: %v", svc.Namespace, svc.Name, err)
			heliosConfig.Status.Phase = balancerv1.StateFailed
			heliosConfig.Status.State = balancerv1.StateFailed
			heliosConfig.Status.Message = err.Error()
			meta.SetStatusCondition(&heliosConfig.Status.Conditions, metav1.Condition{
				Type:               balancerv1.ConditionTypeReady,
				Status:             metav1.ConditionFalse,
				Reason:             balancerv1.ReasonIPAllocationError,
				Message:            err.Error(),
				ObservedGeneration: heliosConfig.Generation,
			})
			meta.SetStatusCondition(&heliosConfig.Status.Conditions, metav1.Condition{
				Type:               balancerv1.ConditionTypeDegraded,
				Status:             metav1.ConditionTrue,
				Reason:             balancerv1.ReasonIPAllocationError,
				Message:            err.Error(),
				ObservedGeneration: heliosConfig.Generation,
			})
			if statusErr := r.Status().Update(ctx, &heliosConfig); statusErr != nil {
				svcLogger.Error(statusErr, "failed to update status after allocation failure")
			}
			r.Metrics.RecordRequeueReason(heliosConfig.Name, heliosConfig.Namespace, "ip_allocation_error")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}

		r.Recorder.Eventf(&heliosConfig, corev1.EventTypeNormal, "IPAllocated",
			"Allocated IP %s to service %s/%s", ip, svc.Namespace, svc.Name)

		// Update HeliosConfig status
		if heliosConfig.Status.AllocatedIPs == nil {
			heliosConfig.Status.AllocatedIPs = make(map[string]string)
		}
		heliosConfig.Status.AllocatedIPs[svc.Name] = ip
		heliosConfig.Status.Phase = balancerv1.StateActive
		heliosConfig.Status.State = balancerv1.StateActive
		heliosConfig.Status.Message = "IP allocated successfully"
		meta.SetStatusCondition(&heliosConfig.Status.Conditions, metav1.Condition{
			Type:               balancerv1.ConditionTypeReady,
			Status:             metav1.ConditionTrue,
			Reason:             balancerv1.ReasonNetworkConfigured,
			Message:            "IP allocated successfully",
			ObservedGeneration: heliosConfig.Generation,
		})
		meta.SetStatusCondition(&heliosConfig.Status.Conditions, metav1.Condition{
			Type:               balancerv1.ConditionTypeDegraded,
			Status:             metav1.ConditionFalse,
			Reason:             balancerv1.ReasonNetworkConfigured,
			Message:            "All allocations healthy",
			ObservedGeneration: heliosConfig.Generation,
		})
		if err := r.Status().Update(ctx, &heliosConfig); err != nil {
			svcLogger.Error(err, "failed to update HeliosConfig status")
			return ctrl.Result{}, err
		}
	}

	r.Metrics.RecordRequeueReason(heliosConfig.Name, heliosConfig.Namespace, "periodic")
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

// handleDeletion handles the deletion of a HeliosConfig
func (r *HeliosConfigReconciler) handleDeletion(ctx context.Context, heliosConfig *balancerv1.HeliosConfig) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithValues(
		LogKeyConfig, heliosConfig.Name,
		LogKeyNamespace, heliosConfig.Namespace,
	)

	if controllerutil.ContainsFinalizer(heliosConfig, heliosConfigFinalizer) {
		allocCount := len(heliosConfig.Status.AllocatedIPs)
		logger.Info("cleaning up allocated IPs", LogKeyAllocatedIPs, allocCount)
		r.Recorder.Eventf(heliosConfig, corev1.EventTypeNormal, "CleanupStarted",
			"Releasing %d allocated IPs", allocCount)

		r.IPMgr.ReleaseAll(ctx, logger, heliosConfig)

		controllerutil.RemoveFinalizer(heliosConfig, heliosConfigFinalizer)
		if err := r.Update(ctx, heliosConfig); err != nil {
			logger.Error(err, "failed to remove finalizer")
			r.Recorder.Eventf(heliosConfig, corev1.EventTypeWarning, "CleanupFailed",
				"Failed to remove finalizer: %v", err)
			return ctrl.Result{}, err
		}
		r.Recorder.Event(heliosConfig, corev1.EventTypeNormal, "CleanupComplete",
			"All allocated IPs released and finalizer removed")
		logger.Info("finalizer removed, deletion complete")
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

// findLoadBalancerServices watches for LoadBalancer type services and enqueues
// all HeliosConfig resources whose IP range covers the service's loadBalancerIP.
func (r *HeliosConfigReconciler) findLoadBalancerServices(ctx context.Context, obj client.Object) []reconcile.Request {
	svc, ok := obj.(*corev1.Service)
	if !ok {
		return nil
	}

	logger := log.FromContext(ctx).WithValues(
		LogKeyService, svc.Name,
		LogKeyNamespace, svc.Namespace,
	)
	var heliosConfigs balancerv1.HeliosConfigList
	if err := r.List(ctx, &heliosConfigs); err != nil {
		logger.Error(err, "failed to list HeliosConfigs in service watch handler")
		return nil
	}

	var requests []reconcile.Request
	for _, hc := range heliosConfigs.Items {
		// If the service has a loadBalancerIP, only enqueue configs whose range covers it.
		// If it has no loadBalancerIP, enqueue all configs so they can attempt allocation.
		if svc.Spec.LoadBalancerIP != "" && !network.IPInRange(svc.Spec.LoadBalancerIP, hc.Spec.IPRange) {
			continue
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      hc.Name,
				Namespace: hc.Namespace,
			},
		})
	}
	return requests
}
