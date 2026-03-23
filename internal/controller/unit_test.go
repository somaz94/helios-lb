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
	"k8s.io/client-go/tools/record"
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
	networkMgr := network.NewNetworkManager()
	metricsRecorder := metrics.NewMetricsRecorder()
	return &HeliosConfigReconciler{
		Client:     cl,
		Scheme:     newTestScheme(),
		NetworkMgr: networkMgr,
		Balancer: loadbalancer.NewLoadBalancer(loadbalancer.BalancerConfig{
			Type: loadbalancer.RoundRobin,
		}),
		Metrics: metricsRecorder,
		IPMgr: &IPManager{
			Client:     cl,
			NetworkMgr: networkMgr,
			Metrics:    metricsRecorder,
		},
		Recorder: record.NewFakeRecorder(100),
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
			IPRange: "192.168.1.100-192.168.1.100",
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

	// Pre-allocate the only IP in range so AllocateIP returns "no available IPs" error
	r.NetworkMgr.AllocateIP("192.168.1.100-192.168.1.100")

	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	// Should return the IP allocation error (status update also fails but IP error comes first)
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

	// HeliosConfig status update error should now be returned
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err == nil {
		t.Fatal("expected error from status update failure")
	}
	if err.Error() != "helios status update error" {
		t.Errorf("expected helios status update error, got: %v", err)
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

func TestFindLoadBalancerServices_MultipleHeliosConfigs(t *testing.T) {
	scheme := newTestScheme()
	helios1 := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "helios-range-1",
			Namespace: "default",
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100-192.168.1.200",
		},
	}
	helios2 := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "helios-range-2",
			Namespace: "default",
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "10.0.0.1-10.0.0.100",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios1, helios2).
		Build()
	r := newTestReconciler(cl)

	// Service with IP in range 1 should only enqueue helios-range-1
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "192.168.1.150",
		},
	}

	requests := r.findLoadBalancerServices(context.Background(), svc)
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].Name != "helios-range-1" {
		t.Errorf("expected helios-range-1, got %s", requests[0].Name)
	}

	// Service with IP in range 2 should only enqueue helios-range-2
	svc2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc-2",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:           corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP: "10.0.0.50",
		},
	}

	requests2 := r.findLoadBalancerServices(context.Background(), svc2)
	if len(requests2) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests2))
	}
	if requests2[0].Name != "helios-range-2" {
		t.Errorf("expected helios-range-2, got %s", requests2[0].Name)
	}

	// Service without loadBalancerIP should enqueue all configs
	svc3 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-svc-3",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeLoadBalancer,
		},
	}

	requests3 := r.findLoadBalancerServices(context.Background(), svc3)
	if len(requests3) != 2 {
		t.Fatalf("expected 2 requests for service without loadBalancerIP, got %d", len(requests3))
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

func TestReconcile_SkipsOtherLBClassServices(t *testing.T) {
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
	metallbClass := "metallb"
	// Service managed by another LB controller (e.g. MetalLB)
	otherLBSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "metallb-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:              corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP:    "192.168.1.100",
			LoadBalancerClass: &metallbClass,
			Ports:             []corev1.ServicePort{{Port: 80}},
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(helios, otherLBSvc).
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

	// Verify the other LB's service was NOT modified
	var svc corev1.Service
	err = cl.Get(context.Background(), types.NamespacedName{Name: "metallb-svc", Namespace: "default"}, &svc)
	if err != nil {
		t.Fatalf("failed to get service: %v", err)
	}
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		t.Error("helios-lb should NOT allocate IP to services with different loadBalancerClass")
	}
	if svc.Annotations != nil {
		if _, ok := svc.Annotations["balancer.helios.dev/load-balancer-class"]; ok {
			t.Error("helios-lb should NOT annotate services with different loadBalancerClass")
		}
	}
}

func TestReconcile_IncludesHeliosLBClassService(t *testing.T) {
	scheme := newTestScheme()
	heliosClass := "helios-lb"
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
			Name:      "helios-svc",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:              corev1.ServiceTypeLoadBalancer,
			LoadBalancerIP:    "192.168.1.100",
			LoadBalancerClass: &heliosClass,
			Ports:             []corev1.ServicePort{{Port: 80}},
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

	// Verify helios-lb class service WAS processed
	var updatedSvc corev1.Service
	err = cl.Get(context.Background(), types.NamespacedName{Name: "helios-svc", Namespace: "default"}, &updatedSvc)
	if err != nil {
		t.Fatalf("failed to get service: %v", err)
	}
	if len(updatedSvc.Status.LoadBalancer.Ingress) == 0 {
		t.Error("helios-lb should allocate IP to services with helios-lb loadBalancerClass")
	}
}

func TestReconcile_IPAllocationError_StatusUpdateSucceeds(t *testing.T) {
	scheme := newTestScheme()
	helios := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange: "192.168.1.100-192.168.1.101",
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

	// Pre-allocate all IPs in the range so AllocateIP returns "no available IPs" error
	r.NetworkMgr.AllocateIP("192.168.1.100-192.168.1.101")
	r.NetworkMgr.AllocateIP("192.168.1.100-192.168.1.101")

	result, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	// Should requeue instead of returning error (retryable error pattern)
	if err != nil {
		t.Fatalf("expected nil error with requeue, got %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("expected RequeueAfter to be set for retryable IP allocation error")
	}

	// Verify the HeliosConfig status was updated to Failed
	var updated balancerv1.HeliosConfig
	if getErr := cl.Get(context.Background(), types.NamespacedName{Name: "test-helios", Namespace: "default"}, &updated); getErr != nil {
		t.Fatalf("failed to get HeliosConfig: %v", getErr)
	}
	if updated.Status.Phase != balancerv1.StateFailed {
		t.Errorf("expected phase Failed, got %s", updated.Status.Phase)
	}
	if updated.Status.Message == "" {
		t.Error("expected non-empty error message in status")
	}
}

func TestHandleDeletion_WithServiceIngressClearing(t *testing.T) {
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
			IPRange: "192.168.1.100-192.168.1.101",
			Method:  "RoundRobin",
		},
		Status: balancerv1.HeliosConfigStatus{
			AllocatedIPs: map[string]string{
				"svc-1": "192.168.1.100",
			},
			Phase: balancerv1.StateActive,
		},
	}

	// Create the service that exists - so the Get in handleDeletion succeeds
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{{Port: 80}},
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
	if result.RequeueAfter != 0 {
		t.Errorf("expected no requeue, got %v", result.RequeueAfter)
	}

	// Verify the service's ingress was cleared
	var updatedSvc corev1.Service
	if getErr := cl.Get(context.Background(), types.NamespacedName{Name: "svc-1", Namespace: "default"}, &updatedSvc); getErr != nil {
		t.Fatalf("failed to get service: %v", getErr)
	}
	if len(updatedSvc.Status.LoadBalancer.Ingress) != 0 {
		t.Error("expected service ingress to be cleared after deletion")
	}
}

func TestHandleDeletion_ServiceIngressClearError(t *testing.T) {
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
			Phase: balancerv1.StateActive,
		},
	}

	// Create the service so Get succeeds
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "svc-1",
			Namespace: "default",
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{{Port: 80}},
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
		WithStatusSubresource(&balancerv1.HeliosConfig{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, c client.Client, subResourceName string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				if _, ok := obj.(*corev1.Service); ok {
					return fmt.Errorf("service status clear error")
				}
				return c.SubResource(subResourceName).Update(ctx, obj, opts...)
			},
		}).
		Build()
	r := newTestReconciler(cl)

	// Should not fail - the error is just logged
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

func TestReconcile_NamespaceSelector(t *testing.T) {
	scheme := newTestScheme()

	heliosConfig := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange:           "10.0.0.1-10.0.0.10",
			NamespaceSelector: []string{"allowed-ns"},
		},
	}

	// Service in allowed namespace - should get IP
	allowedSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc-allowed", Namespace: "allowed-ns"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
	}
	// Service in non-allowed namespace - should be skipped
	blockedSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc-blocked", Namespace: "other-ns"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(heliosConfig, allowedSvc, blockedSvc).
		WithStatusSubresource(heliosConfig, allowedSvc, blockedSvc).
		Build()

	r := newTestReconciler(cl)
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify allowed service got an IP
	var updatedAllowed corev1.Service
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "svc-allowed", Namespace: "allowed-ns"}, &updatedAllowed); err != nil {
		t.Fatalf("failed to get allowed service: %v", err)
	}
	if len(updatedAllowed.Status.LoadBalancer.Ingress) == 0 {
		t.Error("expected allowed service to have an ingress IP")
	}

	// Verify blocked service did NOT get an IP
	var updatedBlocked corev1.Service
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "svc-blocked", Namespace: "other-ns"}, &updatedBlocked); err != nil {
		t.Fatalf("failed to get blocked service: %v", err)
	}
	if len(updatedBlocked.Status.LoadBalancer.Ingress) != 0 {
		t.Error("expected blocked service to NOT have an ingress IP")
	}
}

func TestReconcile_MaxAllocations(t *testing.T) {
	scheme := newTestScheme()

	heliosConfig := &balancerv1.HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "test-helios",
			Namespace:  "default",
			Finalizers: []string{heliosConfigFinalizer},
		},
		Spec: balancerv1.HeliosConfigSpec{
			IPRange:        "10.0.0.1-10.0.0.10",
			MaxAllocations: 1,
		},
	}

	svc1 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc1", Namespace: "default"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
	}
	svc2 := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc2", Namespace: "default"},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(heliosConfig, svc1, svc2).
		WithStatusSubresource(heliosConfig, svc1, svc2).
		Build()

	r := newTestReconciler(cl)

	// First reconcile should allocate 1 IP then stop due to maxAllocations
	_, err := r.Reconcile(context.Background(), reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "test-helios", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check the HeliosConfig status
	var updatedConfig balancerv1.HeliosConfig
	if err := cl.Get(context.Background(), types.NamespacedName{Name: "test-helios", Namespace: "default"}, &updatedConfig); err != nil {
		t.Fatalf("failed to get helios config: %v", err)
	}

	if len(updatedConfig.Status.AllocatedIPs) > 1 {
		t.Errorf("expected at most 1 allocation due to maxAllocations, got %d", len(updatedConfig.Status.AllocatedIPs))
	}
}
