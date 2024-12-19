package loadbalancer

import "github.com/somaz94/helios-lb/internal/metrics"

func (lb *LoadBalancer) GetStats(serviceName string) LoadBalancerStats {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	if stats, exists := lb.stats[serviceName]; exists {
		return *stats
	}
	return LoadBalancerStats{}
}

func (lb *LoadBalancer) UpdateMetrics(recorder metrics.MetricsRecorder) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for serviceName, backends := range lb.backends {
		for _, backend := range backends {
			recorder.RecordBackendHealth(backend.Address, serviceName, backend.IsHealthy())
			recorder.RecordBackendConnections(backend.Address, serviceName, float64(backend.Connections))
		}
	}
}
