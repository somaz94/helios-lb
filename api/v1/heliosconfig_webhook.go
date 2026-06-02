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

	"github.com/somaz94/helios-lb/internal/network"

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
	return nil, v.validateSpec(ctx, hc, "")
}

// ValidateUpdate validates a HeliosConfig on update.
func (v *HeliosConfigValidator) ValidateUpdate(ctx context.Context, _ *HeliosConfig, hc *HeliosConfig) (admission.Warnings, error) {
	helioslog.Info("validate update", "name", hc.Name)
	return nil, v.validateSpec(ctx, hc, hc.Name)
}

// validateSpec runs all field-level and cross-config validations for a HeliosConfig.
// excludeName is the config to exclude from the overlap check (the config itself, on update).
func (v *HeliosConfigValidator) validateSpec(ctx context.Context, hc *HeliosConfig, excludeName string) error {
	if err := validateIPRange(hc.Spec.IPRange); err != nil {
		return fmt.Errorf("ipRange: %w", err)
	}
	if hc.Spec.IPv6Range != "" {
		if err := validateIPRange(hc.Spec.IPv6Range); err != nil {
			return fmt.Errorf("ipv6Range: %w", err)
		}
	}
	if err := validatePorts(hc.Spec.Ports); err != nil {
		return err
	}
	if err := validateWeights(hc.Spec.Weights, hc.Spec.Method); err != nil {
		return err
	}
	if err := validateHealthCheck(hc.Spec.HealthCheck); err != nil {
		return err
	}
	// Cross-config IP range overlap check; skipped when no client is wired (unit tests).
	if v.Client != nil {
		return v.checkIPRangeOverlap(ctx, hc, excludeName)
	}
	return nil
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
	startNorm := network.NormalizeIP(startIP)
	endNorm := network.NormalizeIP(endIP)
	if len(startNorm) != len(endNorm) {
		return fmt.Errorf("mixed IPv4/IPv6 in range %q", ipRange)
	}
	if network.CompareIPs(startNorm, endNorm) > 0 {
		return fmt.Errorf("start IP is greater than end IP in range %q", ipRange)
	}

	return nil
}

// The bound and enum checks in validatePorts, validateWeights, and validateHealthCheck
// intentionally mirror the +kubebuilder:validation markers on the matching spec fields
// (PortConfig.Port/Protocol, WeightConfig.Weight, HealthCheckConfig.Protocol) in
// heliosconfig_types.go. The CRD schema is the primary admission gate; these webhook
// checks are a defense-in-depth backstop and the path unit tests exercise directly.
// Keep the two in lock-step: when a marker bound changes, update the matching check here.

// validatePorts validates port configurations.
func validatePorts(ports []PortConfig) error {
	seen := make(map[int32]bool)
	for _, p := range ports {
		if p.Port < 1 || p.Port > 65535 {
			return fmt.Errorf("port %d out of valid range (1-65535)", p.Port)
		}
		switch p.Protocol {
		case "", "TCP", "UDP":
		default:
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
	switch hc.Protocol {
	case "", "TCP", "HTTP":
	default:
		return fmt.Errorf("invalid health check protocol %q: must be TCP or HTTP", hc.Protocol)
	}
	if hc.Protocol == "HTTP" && hc.HTTPPath == "" {
		return fmt.Errorf("httpPath is required when health check protocol is HTTP")
	}
	return nil
}

// checkIPRangeOverlap checks if the new HeliosConfig's IPv4 and IPv6 ranges overlap
// with any existing HeliosConfig. excludeName is the name of the config to exclude
// from the check (used during updates).
func (v *HeliosConfigValidator) checkIPRangeOverlap(ctx context.Context, hc *HeliosConfig, excludeName string) error {
	var list HeliosConfigList
	if err := v.Client.List(ctx, &list); err != nil {
		// If we can't list, log and skip overlap check rather than blocking.
		helioslog.Error(err, "failed to list HeliosConfigs for overlap check")
		return nil
	}

	if err := overlapForRange(hc.Spec.IPRange, excludeName, list.Items, false); err != nil {
		return err
	}
	if hc.Spec.IPv6Range != "" {
		return overlapForRange(hc.Spec.IPv6Range, excludeName, list.Items, true)
	}
	return nil
}

// overlapForRange reports an error if newRange overlaps the matching IP family range of
// any existing config except excludeName. When useIPv6 is true, each existing config's
// IPv6Range is compared; otherwise its IPRange. Empty or unparseable existing ranges are
// skipped (they are validated on their own admission).
func overlapForRange(newRange, excludeName string, items []HeliosConfig, useIPv6 bool) error {
	newStart, newEnd, err := network.ParseIPRange(newRange)
	if err != nil {
		return nil // already validated by validateIPRange
	}
	for i := range items {
		existing := &items[i]
		if existing.Name == excludeName {
			continue
		}
		exRange := existing.Spec.IPRange
		if useIPv6 {
			exRange = existing.Spec.IPv6Range
		}
		if exRange == "" {
			continue
		}
		existStart, existEnd, err := network.ParseIPRange(exRange)
		if err != nil {
			continue
		}
		// Ranges overlap if newStart <= existEnd && newEnd >= existStart.
		if network.CompareIPs(newStart, existEnd) <= 0 && network.CompareIPs(newEnd, existStart) >= 0 {
			return fmt.Errorf("IP range %q overlaps with existing HeliosConfig %q (range: %s)",
				newRange, existing.Name, exRange)
		}
	}
	return nil
}
