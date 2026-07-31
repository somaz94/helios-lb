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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	balancerv1 "github.com/somaz94/helios-lb/api/v1"
)

// These specs exercise the CRD schema itself (field constraints and CEL
// x-kubernetes-validations) against the envtest apiserver. They cover the
// single-object rules that the validating webhook also checks, so those rules
// hold even when the operator runs without --enable-webhook. Rules needing
// cluster state (the cross-config IP range overlap check) stay in the webhook.
var _ = Describe("HeliosConfig CRD validation", func() {
	var namespace string
	var created []*balancerv1.HeliosConfig
	counter := 0

	BeforeEach(func() {
		counter++
		namespace = fmt.Sprintf("helios-crd-validation-%d", counter)
		Expect(client.IgnoreAlreadyExists(k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}))).To(Succeed())
	})

	AfterEach(func() {
		for _, hc := range created {
			Expect(client.IgnoreNotFound(k8sClient.Delete(ctx, hc))).To(Succeed())
		}
		created = nil
	})

	// newConfig returns a spec that satisfies every CRD rule, so each spec below
	// can mutate exactly the one field under test. Each config gets its own IP so
	// admitted resources never collide.
	newConfig := func(name string) *balancerv1.HeliosConfig {
		return &balancerv1.HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: balancerv1.HeliosConfigSpec{
				IPRange: fmt.Sprintf("10.240.%d.100", counter),
				Method:  methodRoundRobin,
			},
		}
	}

	createOK := func(hc *balancerv1.HeliosConfig) {
		Expect(k8sClient.Create(ctx, hc)).To(Succeed())
		created = append(created, hc)
	}

	Context("weights require the WeightedRoundRobin method", func() {
		It("rejects weights on a non-weighted method", func() {
			hc := newConfig("weights-wrong-method")
			hc.Spec.Weights = []balancerv1.WeightConfig{{ServiceName: nameSvcA, Weight: 10}}

			err := k8sClient.Create(ctx, hc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("weights can only be used with the WeightedRoundRobin method"))
		})

		It("accepts weights with WeightedRoundRobin", func() {
			hc := newConfig("weights-right-method")
			hc.Spec.Method = methodWeighted
			hc.Spec.Weights = []balancerv1.WeightConfig{{ServiceName: nameSvcA, Weight: 10}}

			createOK(hc)
		})

		It("rejects a duplicate serviceName in weights", func() {
			hc := newConfig("weights-duplicate")
			hc.Spec.Method = methodWeighted
			hc.Spec.Weights = []balancerv1.WeightConfig{
				{ServiceName: nameSvcA, Weight: 10},
				{ServiceName: nameSvcA, Weight: 20},
			}

			err := k8sClient.Create(ctx, hc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate serviceName in spec.weights"))
		})

		It("rejects an empty serviceName", func() {
			hc := newConfig("weights-empty-name")
			hc.Spec.Method = methodWeighted
			hc.Spec.Weights = []balancerv1.WeightConfig{{ServiceName: "", Weight: 10}}

			err := k8sClient.Create(ctx, hc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("serviceName"))
		})
	})

	Context("spec.ports uniqueness", func() {
		It("rejects a duplicate port", func() {
			hc := newConfig("ports-duplicate")
			hc.Spec.Ports = []balancerv1.PortConfig{
				{Port: 80, Protocol: protocolTCP},
				{Port: 80, Protocol: "UDP"},
			}

			err := k8sClient.Create(ctx, hc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate port in spec.ports"))
		})

		It("accepts distinct ports", func() {
			hc := newConfig("ports-distinct")
			hc.Spec.Ports = []balancerv1.PortConfig{
				{Port: 80, Protocol: protocolTCP},
				{Port: 443, Protocol: protocolTCP},
			}

			createOK(hc)
		})
	})

	Context("health check httpPath", func() {
		It("rejects HTTP protocol without an httpPath", func() {
			hc := newConfig("healthcheck-no-path")
			hc.Spec.HealthCheck = &balancerv1.HealthCheckConfig{Enabled: true, Protocol: "HTTP"}

			err := k8sClient.Create(ctx, hc)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("httpPath is required when the health check protocol is HTTP"))
		})

		It("accepts HTTP protocol with an httpPath", func() {
			hc := newConfig("healthcheck-with-path")
			hc.Spec.HealthCheck = &balancerv1.HealthCheckConfig{
				Enabled: true, Protocol: "HTTP", HTTPPath: "/healthz",
			}

			createOK(hc)
		})

		It("does not require an httpPath for TCP health checks", func() {
			hc := newConfig("healthcheck-tcp")
			hc.Spec.HealthCheck = &balancerv1.HealthCheckConfig{Enabled: true, Protocol: protocolTCP}

			createOK(hc)
		})
	})

	Context("update path", func() {
		It("rejects an update that violates a CEL rule", func() {
			hc := newConfig("update-guard")
			hc.Spec.Method = methodWeighted
			hc.Spec.Weights = []balancerv1.WeightConfig{{ServiceName: nameSvcA, Weight: 10}}
			createOK(hc)

			// The reconciler writes status right after create, so the local copy goes
			// stale. Re-read inside the retry: updating from the stale copy races
			// into a 409 conflict instead of the CEL rejection this spec asserts.
			Eventually(func(g Gomega) {
				latest := &balancerv1.HeliosConfig{}
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(hc), latest)).To(Succeed())
				latest.Spec.Method = methodRoundRobin

				err := k8sClient.Update(ctx, latest)
				g.Expect(err).To(HaveOccurred())
				g.Expect(err.Error()).To(ContainSubstring("weights can only be used with the WeightedRoundRobin method"))
			}, time.Second*10, time.Millisecond*250).Should(Succeed())
		})
	})
})
