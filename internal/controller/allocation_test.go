package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	balancerv1 "github.com/somaz94/helios-lb/api/v1"
	"github.com/somaz94/helios-lb/internal/metrics"
	"github.com/somaz94/helios-lb/internal/network"
)

const svcWeb = "web"

func lbService(ns, name string, ingress ...string) corev1.Service {
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
	}
	for _, ip := range ingress {
		svc.Status.LoadBalancer.Ingress = append(svc.Status.LoadBalancer.Ingress, corev1.LoadBalancerIngress{IP: ip})
	}
	return svc
}

func TestAllocationKey(t *testing.T) {
	svc := lbService("shop", svcWeb)
	if got := allocationKey(&svc); got != "shop/web" {
		t.Errorf("allocationKey() = %q, want shop/web", got)
	}
	if ns, name := splitAllocationKey("shop/web", "cfg-ns"); ns != "shop" || name != "web" {
		t.Errorf("splitAllocationKey(shop/web) = %s, %s", ns, name)
	}
	if ns, name := splitAllocationKey("web", "cfg-ns"); ns != "cfg-ns" || name != "web" {
		t.Errorf("a legacy key must resolve to the config namespace, got %s, %s", ns, name)
	}
}

func TestMarkServiceIngressReservesAssignedAddresses(t *testing.T) {
	m := newTestIPManager()
	m.MarkServiceIngress([]corev1.Service{
		lbService("a", "one", "192.0.2.1"),
		{Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP}},
	})

	got, err := m.allocateIP("192.0.2.1-192.0.2.2", "")
	if err != nil || got != "192.0.2.2" {
		t.Errorf("allocateIP() = %q, %v; want 192.0.2.2, since .1 is on a service", got, err)
	}
}

func TestReleaseStale(t *testing.T) {
	m := newTestIPManager()
	cfg := &balancerv1.HeliosConfig{ObjectMeta: metav1.ObjectMeta{Namespace: "cfg-ns"}}
	cfg.Status.AllocatedIPs = map[string]string{
		"shop/kept":      "192.0.2.1",
		"shop/deleted":   "192.0.2.2",
		"shop/clusterip": "192.0.2.3",
		"shop/moved":     "192.0.2.4",
		"shop/pending":   "192.0.2.5",
		"legacy":         "192.0.2.6",
	}
	clusterIP := lbService("shop", "clusterip")
	clusterIP.Spec.Type = corev1.ServiceTypeClusterIP
	services := []corev1.Service{
		lbService("shop", "kept", "192.0.2.1"),
		clusterIP,
		lbService("shop", "moved", "192.0.2.99"),
		lbService("shop", "pending"),
		lbService("cfg-ns", "legacy", "192.0.2.6"),
	}

	if !m.ReleaseStale(log.Log, cfg, services) {
		t.Fatal("ReleaseStale() reported no change")
	}
	want := map[string]string{"shop/kept": "192.0.2.1", "shop/pending": "192.0.2.5", "legacy": "192.0.2.6"}
	if fmt.Sprint(cfg.Status.AllocatedIPs) != fmt.Sprint(want) {
		t.Errorf("AllocatedIPs = %v, want %v", cfg.Status.AllocatedIPs, want)
	}
	if m.ReleaseStale(log.Log, cfg, services) {
		t.Error("a second pass with nothing stale reported a change")
	}
}

var _ = Describe("Allocation correctness", func() {
	ctx := context.Background()

	newReconciler := func() *HeliosConfigReconciler {
		networkMgr := network.NewNetworkManager()
		recorder := metrics.NewMetricsRecorder()
		return &HeliosConfigReconciler{
			Client:     k8sClient,
			Scheme:     k8sClient.Scheme(),
			NetworkMgr: networkMgr,
			Metrics:    recorder,
			IPMgr:      &IPManager{Client: k8sClient, NetworkMgr: networkMgr, Metrics: recorder},
			Recorder:   record.NewFakeRecorder(100),
		}
	}
	newNamespace := func(prefix string) string {
		ns := fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})).To(Succeed())
		return ns
	}
	createConfig := func(ns, ipRange string, namespaces ...string) types.NamespacedName {
		key := types.NamespacedName{Namespace: ns, Name: "pool"}
		Expect(k8sClient.Create(ctx, &balancerv1.HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: key.Name, Namespace: ns},
			Spec:       balancerv1.HeliosConfigSpec{IPRange: ipRange, NamespaceSelector: namespaces},
		})).To(Succeed())
		return key
	}
	createService := func(ns, name string) {
		Expect(k8sClient.Create(ctx, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
			Spec: corev1.ServiceSpec{
				Type:  corev1.ServiceTypeLoadBalancer,
				Ports: []corev1.ServicePort{{Port: 80}},
			},
		})).To(Succeed())
	}
	reconcileTwice := func(r *HeliosConfigReconciler, key types.NamespacedName) {
		for range 2 {
			_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).NotTo(HaveOccurred())
		}
	}
	ingressIP := func(ns, name string) string {
		var svc corev1.Service
		Expect(k8sClient.Get(ctx, types.NamespacedName{Namespace: ns, Name: name}, &svc)).To(Succeed())
		if len(svc.Status.LoadBalancer.Ingress) == 0 {
			return ""
		}
		return svc.Status.LoadBalancer.Ingress[0].IP
	}

	It("does not hand out an address again after the controller restarts", func() {
		ns := newNamespace("restart")
		key := createConfig(ns, "198.51.100.10-198.51.100.12", ns)
		createService(ns, "first")
		reconcileTwice(newReconciler(), key)
		first := ingressIP(ns, "first")
		Expect(first).NotTo(BeEmpty())

		// A fresh allocator is what a restarted controller starts with.
		createService(ns, "second")
		reconcileTwice(newReconciler(), key)
		Expect(ingressIP(ns, "second")).NotTo(BeEmpty())
		Expect(ingressIP(ns, "second")).NotTo(Equal(first))
	})

	It("keys allocations by namespace and releases them when the service goes away", func() {
		nsA, nsB := newNamespace("team-a"), newNamespace("team-b")
		key := createConfig(nsA, "198.51.100.20", nsA, nsB)
		createService(nsA, svcWeb)
		createService(nsB, svcWeb)
		r := newReconciler()
		reconcileTwice(r, key)

		var cfg balancerv1.HeliosConfig
		Expect(k8sClient.Get(ctx, key, &cfg)).To(Succeed())
		Expect(cfg.Status.AllocatedIPs).To(HaveLen(1), "a single-IP pool serves one service")
		holder, waiting := nsA, nsB
		if ingressIP(nsA, svcWeb) == "" {
			holder, waiting = nsB, nsA
		}
		Expect(cfg.Status.AllocatedIPs).To(HaveKeyWithValue(holder+"/web", "198.51.100.20"))
		Expect(ingressIP(waiting, svcWeb)).To(BeEmpty())

		Expect(k8sClient.Delete(ctx, &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: holder, Name: svcWeb}})).To(Succeed())
		Eventually(func() int {
			reconcileTwice(r, key)
			Expect(k8sClient.Get(ctx, key, &cfg)).To(Succeed())
			if ingressIP(waiting, svcWeb) == "" {
				return 0
			}
			return len(cfg.Status.AllocatedIPs)
		}, 10*time.Second, 250*time.Millisecond).Should(Equal(1))
		Expect(cfg.Status.AllocatedIPs).To(HaveKeyWithValue(waiting+"/web", "198.51.100.20"))
		Expect(cfg.Status.AllocatedIPs).NotTo(HaveKey(holder + "/web"))
	})
})
