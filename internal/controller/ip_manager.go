package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	balancerv1 "github.com/somaz94/helios-lb/api/v1"
	"github.com/somaz94/helios-lb/internal/metrics"
	"github.com/somaz94/helios-lb/internal/network"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IPManager handles IP allocation, release, and service updates.
type IPManager struct {
	Client     client.Client
	NetworkMgr *network.NetworkManager
	Metrics    *metrics.MetricsRecorder
}

// AllocateAndAssign allocates an IP from the config's range and assigns it to the service.
// It marks IPs from other HeliosConfigs as used to prevent duplicates.
// For dual-stack configs, it allocates both IPv4 and IPv6 addresses.
// Returns the allocated IPv4 IP, IPv6 IP (empty if single-stack), or an error.
func (m *IPManager) AllocateAndAssign(
	ctx context.Context,
	logger logr.Logger,
	heliosConfig *balancerv1.HeliosConfig,
	svc *corev1.Service,
) (string, string, error) {
	// Mark IPs already allocated by other configs to avoid duplicates
	var allConfigs balancerv1.HeliosConfigList
	if err := m.Client.List(ctx, &allConfigs); err != nil {
		return "", "", NewRetryableError("failed to list configs for conflict check", err)
	}
	for _, other := range allConfigs.Items {
		if other.Name == heliosConfig.Name && other.Namespace == heliosConfig.Namespace {
			continue
		}
		for _, ip := range other.Status.AllocatedIPs {
			m.NetworkMgr.MarkUsed(ip)
		}
		for _, ip := range other.Status.AllocatedIPv6s {
			m.NetworkMgr.MarkUsed(ip)
		}
	}

	// Allocate IPv4
	ip, err := m.NetworkMgr.AllocateIP(heliosConfig.Spec.IPRange)
	if err != nil {
		return "", "", NewRetryableError("IPv4 allocation failed", err)
	}

	// Allocate IPv6 if dual-stack
	var ipv6 string
	if heliosConfig.Spec.IPv6Range != "" {
		ipv6, err = m.NetworkMgr.AllocateIP(heliosConfig.Spec.IPv6Range)
		if err != nil {
			m.NetworkMgr.ReleaseIP(ip)
			return "", "", NewRetryableError("IPv6 allocation failed", err)
		}
	}

	svcLogger := logger.WithValues(LogKeyIP, ip, LogKeyService, svc.Name)
	if ipv6 != "" {
		svcLogger = svcLogger.WithValues(LogKeyIPv6, ipv6)
	}

	if err := m.assignIPToService(ctx, svc, ip, ipv6); err != nil {
		m.NetworkMgr.ReleaseIP(ip)
		if ipv6 != "" {
			m.NetworkMgr.ReleaseIP(ipv6)
		}
		return "", "", NewRetryableError("service update failed", err)
	}

	m.Metrics.RecordIPAllocation(ip, true)
	if ipv6 != "" {
		m.Metrics.RecordIPAllocation(ipv6, true)
		svcLogger.Info("dual-stack IPs allocated and assigned to service")
	} else {
		svcLogger.Info("IP allocated and assigned to service")
	}
	return ip, ipv6, nil
}

// assignIPToService updates the service with the allocated IPs using retry on conflict.
// For dual-stack, both IPv4 and IPv6 are added to the ingress list.
func (m *IPManager) assignIPToService(ctx context.Context, svc *corev1.Service, ip string, ipv6 string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		var currentSvc corev1.Service
		if err := m.Client.Get(ctx, types.NamespacedName{
			Namespace: svc.Namespace,
			Name:      svc.Name,
		}, &currentSvc); err != nil {
			return err
		}

		if currentSvc.Annotations == nil {
			currentSvc.Annotations = make(map[string]string)
		}
		currentSvc.Annotations["balancer.helios.dev/load-balancer-class"] = balancerv1.LoadBalancerClassHelios
		currentSvc.Spec.LoadBalancerClass = pointer.String(balancerv1.LoadBalancerClassHelios)

		ingress := []corev1.LoadBalancerIngress{{IP: ip}}
		if ipv6 != "" {
			ingress = append(ingress, corev1.LoadBalancerIngress{IP: ipv6})
		}
		currentSvc.Status.LoadBalancer.Ingress = ingress

		if err := m.Client.Status().Update(ctx, &currentSvc); err != nil {
			return err
		}
		return m.Client.Update(ctx, &currentSvc)
	})
}

// ReleaseAll releases all allocated IPs (IPv4 and IPv6) and clears service ingress for a config.
func (m *IPManager) ReleaseAll(
	ctx context.Context,
	logger logr.Logger,
	heliosConfig *balancerv1.HeliosConfig,
) {
	// Collect all service names that need ingress cleared
	serviceNames := make(map[string]bool)

	for serviceName, ip := range heliosConfig.Status.AllocatedIPs {
		m.NetworkMgr.ReleaseIP(ip)
		m.Metrics.RecordIPAllocation(ip, false)
		logger.Info("released IPv4", LogKeyService, serviceName, LogKeyIP, ip)
		serviceNames[serviceName] = true
	}

	for serviceName, ip := range heliosConfig.Status.AllocatedIPv6s {
		m.NetworkMgr.ReleaseIP(ip)
		m.Metrics.RecordIPAllocation(ip, false)
		logger.Info("released IPv6", LogKeyService, serviceName, LogKeyIPv6, ip)
		serviceNames[serviceName] = true
	}

	// Clear ingress for all affected services
	for serviceName := range serviceNames {
		var svc corev1.Service
		if err := m.Client.Get(ctx, types.NamespacedName{
			Name:      serviceName,
			Namespace: heliosConfig.Namespace,
		}, &svc); err == nil {
			svc.Status.LoadBalancer.Ingress = nil
			if err := m.Client.Status().Update(ctx, &svc); err != nil {
				logger.Error(err, "failed to clear service ingress",
					LogKeyService, serviceName)
			}
		}
	}
}

// CheckIPConflicts checks if any IP in this config's ranges (IPv4 and IPv6) is already allocated by another HeliosConfig.
// Returns a map of conflicting IPs to the owning HeliosConfig name, or nil if no conflicts.
func (m *IPManager) CheckIPConflicts(
	ctx context.Context,
	heliosConfig *balancerv1.HeliosConfig,
) (map[string]string, error) {
	var allConfigs balancerv1.HeliosConfigList
	if err := m.Client.List(ctx, &allConfigs); err != nil {
		return nil, fmt.Errorf("failed to list HeliosConfigs: %w", err)
	}

	conflicts := make(map[string]string)
	for _, other := range allConfigs.Items {
		if other.Name == heliosConfig.Name && other.Namespace == heliosConfig.Namespace {
			continue
		}
		// Check IPv4 conflicts
		for svcName, ip := range other.Status.AllocatedIPs {
			if network.IPInRange(ip, heliosConfig.Spec.IPRange) {
				conflicts[ip] = fmt.Sprintf("%s/%s (svc: %s)", other.Namespace, other.Name, svcName)
			}
		}
		// Check IPv6 conflicts
		if heliosConfig.Spec.IPv6Range != "" {
			for svcName, ip := range other.Status.AllocatedIPv6s {
				if network.IPInRange(ip, heliosConfig.Spec.IPv6Range) {
					conflicts[ip] = fmt.Sprintf("%s/%s (svc: %s, ipv6)", other.Namespace, other.Name, svcName)
				}
			}
		}
	}

	if len(conflicts) == 0 {
		return nil, nil
	}
	return conflicts, nil
}

// CheckQuota returns an error if the config has reached its max allocations.
func (m *IPManager) CheckQuota(heliosConfig *balancerv1.HeliosConfig) error {
	if heliosConfig.Spec.MaxAllocations > 0 &&
		int32(len(heliosConfig.Status.AllocatedIPs)) >= heliosConfig.Spec.MaxAllocations {
		return fmt.Errorf("max allocations reached: %d/%d",
			len(heliosConfig.Status.AllocatedIPs), heliosConfig.Spec.MaxAllocations)
	}
	return nil
}
