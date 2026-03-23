package controller

import (
	"github.com/somaz94/helios-lb/internal/network"
	corev1 "k8s.io/api/core/v1"
)

// FilterEligibleServices returns LoadBalancer services that should be managed by the given config.
func FilterEligibleServices(services []corev1.Service, namespaceSelector []string, ipRange string) []corev1.Service {
	nsAllowed := make(map[string]bool, len(namespaceSelector))
	for _, ns := range namespaceSelector {
		nsAllowed[ns] = true
	}

	var result []corev1.Service
	for _, svc := range services {
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			continue
		}
		if svc.Spec.LoadBalancerClass != nil && *svc.Spec.LoadBalancerClass != "helios-lb" {
			continue
		}
		if len(nsAllowed) > 0 && !nsAllowed[svc.Namespace] {
			continue
		}
		// Skip services that already have an ingress IP assigned
		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			continue
		}
		// If loadBalancerIP is set, it must fall within the config's IP range
		if svc.Spec.LoadBalancerIP != "" && !network.IPInRange(svc.Spec.LoadBalancerIP, ipRange) {
			continue
		}
		result = append(result, svc)
	}
	return result
}
