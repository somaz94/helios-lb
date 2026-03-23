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

package v1

import (
	"context"
	"fmt"
	"net"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var helioslog = logf.Log.WithName("heliosconfig-webhook")

// HeliosConfigValidator implements admission.Validator[*HeliosConfig].
// +kubebuilder:object:generate=false
type HeliosConfigValidator struct {
	Client client.Reader
}

// SetupWebhookWithManager registers the validating webhook with the manager.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &HeliosConfig{}).
		WithValidator(&HeliosConfigValidator{Client: mgr.GetClient()}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-balancer-helios-dev-v1-heliosconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=balancer.helios.dev,resources=heliosconfigs,verbs=create;update,versions=v1,name=vheliosconfig.kb.io,admissionReviewVersions=v1

var _ admission.Validator[*HeliosConfig] = &HeliosConfigValidator{}

// ValidateCreate validates a HeliosConfig on creation.
func (v *HeliosConfigValidator) ValidateCreate(ctx context.Context, hc *HeliosConfig) (admission.Warnings, error) {
	helioslog.Info("validate create", "name", hc.Name)

	if err := validateIPRange(hc.Spec.IPRange); err != nil {
		return nil, err
	}
	if err := validatePorts(hc.Spec.Ports); err != nil {
		return nil, err
	}
	if err := validateWeights(hc.Spec.Weights, hc.Spec.Method); err != nil {
		return nil, err
	}
	if err := validateHealthCheck(hc.Spec.HealthCheck); err != nil {
		return nil, err
	}

	// Check for IP range overlap with existing HeliosConfigs
	if v.Client != nil {
		if err := v.checkIPRangeOverlap(ctx, hc, ""); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// ValidateUpdate validates a HeliosConfig on update.
func (v *HeliosConfigValidator) ValidateUpdate(ctx context.Context, _ *HeliosConfig, hc *HeliosConfig) (admission.Warnings, error) {
	helioslog.Info("validate update", "name", hc.Name)

	if err := validateIPRange(hc.Spec.IPRange); err != nil {
		return nil, err
	}
	if err := validatePorts(hc.Spec.Ports); err != nil {
		return nil, err
	}
	if err := validateWeights(hc.Spec.Weights, hc.Spec.Method); err != nil {
		return nil, err
	}
	if err := validateHealthCheck(hc.Spec.HealthCheck); err != nil {
		return nil, err
	}

	// Check overlap excluding self
	if v.Client != nil {
		if err := v.checkIPRangeOverlap(ctx, hc, hc.Name); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// ValidateDelete validates a HeliosConfig on deletion.
func (v *HeliosConfigValidator) ValidateDelete(_ context.Context, _ *HeliosConfig) (admission.Warnings, error) {
	return nil, nil
}

// validateIPRange validates the IP range format.
func validateIPRange(ipRange string) error {
	if ipRange == "" {
		return fmt.Errorf("ipRange is required")
	}

	trimmed := strings.TrimSpace(ipRange)

	// CIDR notation
	if strings.Contains(trimmed, "/") {
		_, _, err := net.ParseCIDR(trimmed)
		if err != nil {
			return fmt.Errorf("invalid CIDR format %q: %w", ipRange, err)
		}
		return nil
	}

	// Single IP
	if ip := net.ParseIP(trimmed); ip != nil {
		return nil
	}

	// Range format
	parts := strings.Split(trimmed, "-")
	if len(parts) != 2 {
		return fmt.Errorf("invalid IP range format %q: expected single IP, CIDR, or range (start-end)", ipRange)
	}

	startIP := net.ParseIP(strings.TrimSpace(parts[0]))
	endIP := net.ParseIP(strings.TrimSpace(parts[1]))

	if startIP == nil {
		return fmt.Errorf("invalid start IP in range %q", ipRange)
	}
	if endIP == nil {
		return fmt.Errorf("invalid end IP in range %q", ipRange)
	}

	// Ensure start <= end
	startNorm := normalizeIPForValidation(startIP)
	endNorm := normalizeIPForValidation(endIP)
	if len(startNorm) != len(endNorm) {
		return fmt.Errorf("mixed IPv4/IPv6 in range %q", ipRange)
	}
	for i := range startNorm {
		if startNorm[i] < endNorm[i] {
			break
		}
		if startNorm[i] > endNorm[i] {
			return fmt.Errorf("start IP is greater than end IP in range %q", ipRange)
		}
	}

	return nil
}

// normalizeIPForValidation returns consistent IP representation for comparison.
func normalizeIPForValidation(ip net.IP) net.IP {
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip.To16()
}

// validatePorts validates port configurations.
func validatePorts(ports []PortConfig) error {
	seen := make(map[int32]bool)
	for _, p := range ports {
		if p.Port < 1 || p.Port > 65535 {
			return fmt.Errorf("port %d out of valid range (1-65535)", p.Port)
		}
		if p.Protocol != "" && p.Protocol != "TCP" && p.Protocol != "UDP" {
			return fmt.Errorf("invalid protocol %q for port %d: must be TCP or UDP", p.Protocol, p.Port)
		}
		if seen[p.Port] {
			return fmt.Errorf("duplicate port %d", p.Port)
		}
		seen[p.Port] = true
	}
	return nil
}

// validateWeights validates weight configurations.
func validateWeights(weights []WeightConfig, method string) error {
	if len(weights) > 0 && method != "WeightedRoundRobin" {
		return fmt.Errorf("weights can only be used with WeightedRoundRobin method, got %q", method)
	}
	seen := make(map[string]bool)
	for _, w := range weights {
		if w.ServiceName == "" {
			return fmt.Errorf("weight serviceName is required")
		}
		if w.Weight < 1 || w.Weight > 100 {
			return fmt.Errorf("weight for service %q must be between 1 and 100, got %d", w.ServiceName, w.Weight)
		}
		if seen[w.ServiceName] {
			return fmt.Errorf("duplicate weight for service %q", w.ServiceName)
		}
		seen[w.ServiceName] = true
	}
	return nil
}

// validateHealthCheck validates health check configuration.
func validateHealthCheck(hc *HealthCheckConfig) error {
	if hc == nil {
		return nil
	}
	if hc.Protocol != "" && hc.Protocol != "TCP" && hc.Protocol != "HTTP" {
		return fmt.Errorf("invalid health check protocol %q: must be TCP or HTTP", hc.Protocol)
	}
	if hc.Protocol == "HTTP" && hc.HTTPPath == "" {
		return fmt.Errorf("httpPath is required when health check protocol is HTTP")
	}
	return nil
}

// checkIPRangeOverlap checks if the new HeliosConfig's IP range overlaps
// with any existing HeliosConfig. excludeName is the name of the config to
// exclude from the check (used during updates).
func (v *HeliosConfigValidator) checkIPRangeOverlap(ctx context.Context, hc *HeliosConfig, excludeName string) error {
	var list HeliosConfigList
	if err := v.Client.List(ctx, &list); err != nil {
		// If we can't list, log and skip overlap check rather than blocking
		helioslog.Error(err, "failed to list HeliosConfigs for overlap check")
		return nil
	}

	newStart, newEnd, err := parseRange(hc.Spec.IPRange)
	if err != nil {
		return nil // already validated
	}

	for _, existing := range list.Items {
		if existing.Name == excludeName {
			continue
		}
		existStart, existEnd, err := parseRange(existing.Spec.IPRange)
		if err != nil {
			continue
		}
		// Check overlap: ranges overlap if newStart <= existEnd && newEnd >= existStart
		if compareIPs(newStart, existEnd) <= 0 && compareIPs(newEnd, existStart) >= 0 {
			return fmt.Errorf("IP range %q overlaps with existing HeliosConfig %q (range: %s)",
				hc.Spec.IPRange, existing.Name, existing.Spec.IPRange)
		}
	}
	return nil
}

// parseRange parses an IP range string and returns normalized start/end IPs.
func parseRange(ipRange string) (start, end net.IP, err error) {
	trimmed := strings.TrimSpace(ipRange)

	if strings.Contains(trimmed, "/") {
		_, ipNet, err := net.ParseCIDR(trimmed)
		if err != nil {
			return nil, nil, err
		}
		s := normalizeIPForValidation(ipNet.IP)
		e := make(net.IP, len(s))
		for i := range s {
			e[i] = s[i] | ^ipNet.Mask[i]
		}
		return s, e, nil
	}

	if ip := net.ParseIP(trimmed); ip != nil {
		n := normalizeIPForValidation(ip)
		return n, n, nil
	}

	parts := strings.Split(trimmed, "-")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid range: %s", ipRange)
	}

	s := net.ParseIP(strings.TrimSpace(parts[0]))
	e := net.ParseIP(strings.TrimSpace(parts[1]))
	if s == nil || e == nil {
		return nil, nil, fmt.Errorf("invalid IPs in range: %s", ipRange)
	}
	return normalizeIPForValidation(s), normalizeIPForValidation(e), nil
}

// compareIPs compares two IPs byte-by-byte. Returns -1, 0, or 1.
func compareIPs(a, b net.IP) int {
	a = normalizeIPForValidation(a)
	b = normalizeIPForValidation(b)
	if len(a) != len(b) {
		if len(a) < len(b) {
			return -1
		}
		return 1
	}
	for i := range a {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}
