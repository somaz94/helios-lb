package controller

import (
	"context"
	"fmt"
	"testing"

	balancerv1 "github.com/somaz94/helios-lb/api/v1"
	"github.com/somaz94/helios-lb/internal/loadbalancer"
	"github.com/somaz94/helios-lb/internal/metrics"
	"github.com/somaz94/helios-lb/internal/network"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = balancerv1.AddToScheme(s)
	return s
}

func newTestReconciler(cl client.Client) *HeliosConfigReconciler {
	return &HeliosConfigReconciler{
		Client:     cl,
		Scheme:     newTestScheme(),
		NetworkMgr: network.NewNetworkManager(),
		Balancer: loadbalancer.NewLoadBalancer(loadbalancer.BalancerConfig{
			Type: loadbalancer.RoundRobin,
		}),
		Metrics: metrics.NewMetricsRecorder(),
	}
}

func TestReconcile_NotFound(t *testing.T) {
	cl := fake.NewClientBuilder().
		WithScheme(newTestScheme()).
		Build()
	r := newTestReconciler(cl)

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got %v", result.RequeueAfter)
	}
}

func TestReconcile_GetError(t *testing.T) {
	cl := fake.NewClientBuilder().
		WithScheme(newTestScheme()).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				return fmt.Errorf("get error")
			},
		}).
		Build()
	r := newTestReconciler(cl)

	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test", Namespace: "default"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReconcile_AddFinalizerError(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-helios",
			Namespace: "default",
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios).
		WithStatusSubresource(&balancerv1.HeliosConfig{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if _, ok := obj.(*balancerv1.HeliosConfig); ok {
					return fmt.Errorf("update error")
				}
				return c.Update(ctx, obj, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err == nil {
		t.Fatal("expected error from finalizer update, got nil")
	}
}

func TestReconcile_ListServicesError(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios).
		WithStatusSubresource(&balancerv1.HeliosConfig{}).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				if _, ok := list.(*corev1.ServiceList); ok {
					return fmt.Errorf("list services error")
				}
				return c.List(ctx, list, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err == nil {
		t.Fatal("expected error from list services, got nil")
	}
}

func TestReconcile_IPAllocationError_StatusUpdateFails(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "invalid-ip",
			Method:  "RoundRobin",
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "invalid-ip",
			Ports:          []corev1.ServicePort{{Port: 80}},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, svc).
		WithStatusSubresource(&balancerv1.HeliosConfig{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*balancerv1.HeliosConfig); ok {
					return fmt.Errorf("status update error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	// Should still return the IP allocation error
	if err == nil {
		t.Fatal("expected error from IP allocation, got nil")
	}
}

func TestReconcile_ServiceUpdateRetryError(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "192.168.1.100",
			Ports:          []corev1.ServicePort{{Port: 80}},
		},
	}

	svcGetCount := 0
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, svc).
		WithStatusSubresource(&balancerv1.HeliosConfig{}, &corev1.Service{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*corev1.Service); ok {
					svcGetCount++
					// Inside retry callback, Get for service should fail
					return fmt.Errorf("get service error in retry")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	// Should not return error because retry failure causes continue
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected requeue after")
	}
}

func TestReconcile_ServiceStatusUpdateError(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "192.168.1.100",
			Ports:          []corev1.ServicePort{{Port: 80}},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, svc).
		WithStatusSubresource(&balancerv1.HeliosConfig{}, &corev1.Service{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*corev1.Service); ok {
					return fmt.Errorf("service status update error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	// Service status update failure in retry causes continue, not error
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected requeue after")
	}
}

func TestReconcile_HeliosStatusUpdateError(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "192.168.1.100",
			Ports:          []corev1.ServicePort{{Port: 80}},
		},
	}

	statusUpdateCount := 0
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, svc).
		WithStatusSubresource(&balancerv1.HeliosConfig{}, &corev1.Service{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*balancerv1.HeliosConfig); ok {
					statusUpdateCount++
					// Let service status updates succeed, fail helios status update
					return fmt.Errorf("helios status update error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	// HeliosConfig status update error is logged but not returned
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected requeue after")
	}
}

func TestHandleDeletion_WithAllocatedIPs(t *testing.T) {
	scheme := newTestScheme()
	now := metav1.Now()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-helios",
			Namespace:         "default",
			Finalizers:        []string{heliosConfigFinalizer},
			DeletionTimestamp: &now,
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
		Status: balancerv1.HeliosConfigStatus{
			AllocatedIPs: map[string]string{
				"svc-1": "192.168.1.100",
				"svc-2": "192.168.1.101",
			},
			Phase: balancerv1.StateActive,
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios).
		WithStatusSubresource(&balancerv1.HeliosConfig{}).
		Build()
	r := newTestReconciler(cl)

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got %v", result.RequeueAfter)
	}
}

func TestHandleDeletion_RemoveFinalizerError(t *testing.T) {
	scheme := newTestScheme()
	now := metav1.Now()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-helios",
			Namespace:         "default",
			Finalizers:        []string{heliosConfigFinalizer},
			DeletionTimestamp: &now,
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
		Status: balancerv1.HeliosConfigStatus{
			AllocatedIPs: map[string]string{
				"svc-1": "192.168.1.100",
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios).
		WithStatusSubresource(&balancerv1.HeliosConfig{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if hc, ok := obj.(*balancerv1.HeliosConfig); ok {
					if !controllerutil.ContainsFinalizer(hc, heliosConfigFinalizer) {
						return fmt.Errorf("remove finalizer error")
					}
				}
				return c.Update(ctx, obj, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err == nil {
		t.Fatal("expected error from remove finalizer, got nil")
	}
}

func TestHandleDeletion_NoFinalizer(t *testing.T) {
	scheme := newTestScheme()
	now := metav1.Now()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "test-helios",
			Namespace:         "default",
			DeletionTimestamp: &now,
			Finalizers:        []string{"other-finalizer"},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios).
		WithStatusSubresource(&balancerv1.HeliosConfig{}).
		Build()
	r := newTestReconciler(cl)

	result, err := r.handleDeletion(context.Background(), helios)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got %v", result.RequeueAfter)
	}
}

func TestFindLoadBalancerServices_WithService(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-helios",
			Namespace: "default",
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios).
		Build()
	r := newTestReconciler(cl)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
		},
	}

	requests := r.findLoadBalancerServices(context.Background(), svc)
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Name != "test-helios" {
		t.Errorf("expected request for test-helios, got %s", requests[0].Name)
	}
}

func TestFindLoadBalancerServices_NoHeliosConfigs(t *testing.T) {
	scheme := newTestScheme()
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	r := newTestReconciler(cl)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
	}

	requests := r.findLoadBalancerServices(context.Background(), svc)
	if len(requests) != 0 {
		t.Fatalf("expected 0 requests, got %d", len(requests))
	}
}

func TestFindLoadBalancerServices_NotAService(t *testing.T) {
	scheme := newTestScheme()
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()
	r := newTestReconciler(cl)

	// Pass a non-Service object
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	requests := r.findLoadBalancerServices(context.Background(), pod)
	if requests != nil {
		t.Fatalf("expected nil, got %v", requests)
	}
}

func TestFindLoadBalancerServices_ListError(t *testing.T) {
	scheme := newTestScheme()
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, c client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				return fmt.Errorf("list error")
			},
		}).
		Build()
	r := newTestReconciler(cl)

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
	}

	requests := r.findLoadBalancerServices(context.Background(), svc)
	if requests != nil {
		t.Fatalf("expected nil on list error, got %v", requests)
	}
}

func TestReconcile_SuccessfulIPAllocation(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "192.168.1.100",
			Ports:          []corev1.ServicePort{{Port: 80}},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, svc).
		WithStatusSubresource(&balancerv1.HeliosConfig{}, &corev1.Service{}).
		Build()
	r := newTestReconciler(cl)

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected requeue after")
	}

	// Verify service got IP
	var updatedSvc corev1.Service
	err = cl.Get(context.Background(), types.NamespacedName{Name: "test-svc", Namespace: "default"}, &updatedSvc)
	if err != nil {
		t.Fatalf("failed to get service: %v", err)
	}
	if len(updatedSvc.Status.LoadBalancer.Ingress) == 0 {
		t.Error("expected service to have ingress IP")
	}
}

func TestReconcile_NoLBServices(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}
	// ClusterIP service - not LB
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{{Port: 80}},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, svc).
		WithStatusSubresource(&balancerv1.HeliosConfig{}).
		Build()
	r := newTestReconciler(cl)

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected requeue after")
	}
}

func TestReconcile_ServiceAlreadyHasIngress(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "192.168.1.100",
			Ports:          []corev1.ServicePort{{Port: 80}},
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{{IP: "192.168.1.100"}},
			},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, svc).
		WithStatusSubresource(&balancerv1.HeliosConfig{}, &corev1.Service{}).
		Build()
	r := newTestReconciler(cl)

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected requeue after")
	}
}

func TestReconcile_ServiceUpdateError(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100",
			Method:  "RoundRobin",
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "192.168.1.100",
			Ports:          []corev1.ServicePort{{Port: 80}},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, svc).
		WithStatusSubresource(&balancerv1.HeliosConfig{}, &corev1.Service{}).
		WithInterceptorFuncs(interceptor.Funcs{
			Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				if _, ok := obj.(*corev1.Service); ok {
					return fmt.Errorf("service update error")
				}
				return c.Update(ctx, obj, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	// Service update error in retry causes continue, should not fail reconcile
	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected requeue after")
	}
}
