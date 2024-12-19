package loadbalancer

import "sync/atomic"

// IsHealthy returns the current health status of the backend
func (b *Backend) IsHealthy() bool {
	return atomic.LoadInt32(&b.healthy) == 1
}

// SetHealthy sets the health status of the backend
func (b *Backend) SetHealthy(healthy bool) {
	if healthy {
		atomic.StoreInt32(&b.healthy, 1)
	} else {
		atomic.StoreInt32(&b.healthy, 0)
	}
}

// AddBackend adds a new backend server
func (lb *LoadBalancer) AddBackend(backend *Backend) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if _, exists := lb.backends[backend.ServiceName]; !exists {
		lb.backends[backend.ServiceName] = make([]*Backend, 0)
		lb.stats[backend.ServiceName] = &LoadBalancerStats{}
	}
	lb.backends[backend.ServiceName] = append(lb.backends[backend.ServiceName], backend)
}

// RemoveBackend removes a backend server
func (lb *LoadBalancer) RemoveBackend(address string, serviceName string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if backends, exists := lb.backends[serviceName]; exists {
		for i, backend := range backends {
			if backend.Address == address {
				lb.backends[serviceName] = append(backends[:i], backends[i+1:]...)
				break
			}
		}
	}
}
