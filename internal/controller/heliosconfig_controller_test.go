package controller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	balancerv1 "github.com/somaz94/helios-lb/api/v1"
	"github.com/somaz94/helios-lb/internal/loadbalancer"
	"github.com/somaz94/helios-lb/internal/metrics"
	"github.com/somaz94/helios-lb/internal/network"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("HeliosConfig Controller", func() {
	const namespace = "default"
	var testID int

	ctx := context.Background()
	var reconciler *HeliosConfigReconciler

	BeforeEach(func() {
		testID++
		networkMgr := network.NewNetworkManager()
		balancer := loadbalancer.NewLoadBalancer(loadbalancer.BalancerConfig{
			Type:           loadbalancer.RoundRobin,
			HealthCheck:    true,
			CheckInterval:  time.Second * 5,
			MetricsEnabled: true,
		})
		metricsRecorder := metrics.NewMetricsRecorder()

		reconciler = &HeliosConfigReconciler{
			Client:     k8sClient,
			Scheme:     k8sClient.Scheme(),
			NetworkMgr: networkMgr,
			Balancer:   balancer,
			Metrics:    metricsRecorder,
		}
	})

	It("should allocate IP to LoadBalancer service", func() {
		resourceName := fmt.Sprintf("test-helios-%d", testID)
		serviceName := fmt.Sprintf("test-service-%d", testID)
		namespacedName := types.NamespacedName{Name: resourceName, Namespace: namespace}

		By("Creating a new HeliosConfig")
		heliosConfig := &balancerv1.HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: balancerv1.HeliosConfigSpec{
				IPRange: "192.168.1.100-192.168.1.200",
				Method:  "RoundRobin",
			},
		}
		Expect(k8sClient.Create(ctx, heliosConfig)).To(Succeed())

		By("Creating a LoadBalancer service")
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Type:  corev1.ServiceTypeLoadBalancer,
				Ports: []corev1.ServicePort{{Port: 80}},
			},
		}
		Expect(k8sClient.Create(ctx, service)).To(Succeed())

		By("Reconciling the request")
		req := reconcile.Request{NamespacedName: namespacedName}
		result, err := reconciler.Reconcile(ctx, req)
		Expect(err).NotTo(HaveOccurred())
		Expect(result.RequeueAfter).To(Equal(5 * time.Second))

		By("Verifying the service gets an IP address")
		var svc corev1.Service
		err = k8sClient.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, &svc)
		Expect(err).NotTo(HaveOccurred())
		Expect(len(svc.Status.LoadBalancer.Ingress)).To(BeNumerically(">", 0))
	})

	It("should handle invalid IP range", func() {
		resourceName := fmt.Sprintf("test-helios-%d", testID)
		serviceName := fmt.Sprintf("test-service-%d", testID)
		namespacedName := types.NamespacedName{Name: resourceName, Namespace: namespace}

		By("Creating a HeliosConfig with invalid IP range")
		heliosConfig := &balancerv1.HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: balancerv1.HeliosConfigSpec{
				IPRange: "invalid-range",
				Method:  "RoundRobin",
			},
		}
		Expect(k8sClient.Create(ctx, heliosConfig)).To(Succeed())

		By("Creating a LoadBalancer service")
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      serviceName,
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Type:  corev1.ServiceTypeLoadBalancer,
				Ports: []corev1.ServicePort{{Port: 80}},
			},
		}
		Expect(k8sClient.Create(ctx, service)).To(Succeed())

		By("Reconciling the request")
		req := reconcile.Request{NamespacedName: namespacedName}
		_, err := reconciler.Reconcile(ctx, req)
		Expect(err).To(HaveOccurred())

		By("Verifying the HeliosConfig status")
		err = k8sClient.Get(ctx, namespacedName, heliosConfig)
		Expect(err).NotTo(HaveOccurred())
		Expect(heliosConfig.Status.Phase).To(Equal("Failed"))
	})

	It("should handle deletion with cleanup", func() {
		resourceName := fmt.Sprintf("test-helios-%d", testID)
		namespacedName := types.NamespacedName{Name: resourceName, Namespace: namespace}

		By("Creating a new HeliosConfig")
		heliosConfig := &balancerv1.HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: balancerv1.HeliosConfigSpec{
				IPRange: "192.168.1.100-192.168.1.200",
				Method:  "RoundRobin",
			},
		}
		Expect(k8sClient.Create(ctx, heliosConfig)).To(Succeed())

		By("Deleting the HeliosConfig")
		Expect(k8sClient.Delete(ctx, heliosConfig)).To(Succeed())

		By("Confirming the HeliosConfig is deleted")
		err := k8sClient.Get(ctx, namespacedName, heliosConfig)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})
})
