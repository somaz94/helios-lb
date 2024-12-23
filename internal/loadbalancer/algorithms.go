package loadbalancer

import (
	"hash/fnv"
	"math/rand"
	"sync/atomic"
)

// NextBackend returns the next backend server based on the balancing method
func (lb *LoadBalancer) NextBackend(serviceName string, clientIP string) *Backend {
	lb.mu.RLock()
	backends, exists := lb.backends[serviceName]
	if !exists || len(backends) == 0 {
		lb.mu.RUnlock()
		return nil
	}

	backendsCopy := make([]*Backend, len(backends))
	copy(backendsCopy, backends)
	lb.mu.RUnlock()

	switch lb.config.Type {
	case LeastConnection:
		return lb.leastConnectionBackend(backendsCopy)
	case WeightedRoundRobin:
		return lb.weightedRoundRobinBackend(backendsCopy)
	case IPHash:
		return lb.ipHashBackend(backendsCopy, clientIP)
	case RandomSelection:
		return lb.randomBackend(backendsCopy)
	default: // RoundRobin
		return lb.roundRobinBackend(backendsCopy)
	}
}

// roundRobinBackend implements round-robin selection
func (lb *LoadBalancer) roundRobinBackend(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	serviceName := backends[0].ServiceName

	// rrStates에 대한 초기화 및 ��근을 분리
	lb.mu.Lock()
	if _, exists := lb.rrStates[serviceName]; !exists {
		lb.rrStates[serviceName] = &RoundRobinState{}
	}
	state := lb.rrStates[serviceName]
	lb.mu.Unlock()

	// atomic 연산으로 안전하게 처리
	index := atomic.AddUint32(&state.current, 1) % uint32(len(backends))
	return backends[index]
}

// leastConnectionBackend implements least-connection selection
func (lb *LoadBalancer) leastConnectionBackend(backends []*Backend) *Backend {
	var leastConnBackend *Backend
	leastConn := int32(^uint32(0) >> 1)

	for _, backend := range backends {
		if !backend.IsHealthy() {
			continue
		}
		connections := atomic.LoadInt32(&backend.Connections)
		if connections < leastConn {
			leastConn = connections
			leastConnBackend = backend
		}
	}

	return leastConnBackend
}

// Weighted Round Robin
func (lb *LoadBalancer) weightedRoundRobinBackend(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	serviceName := backends[0].ServiceName
	lb.mu.Lock()
	if _, exists := lb.rrStates[serviceName]; !exists {
		lb.rrStates[serviceName] = &RoundRobinState{}
	}
	state := lb.rrStates[serviceName]
	lb.mu.Unlock()

	// Filter only healthy backend and apply weight
	var healthyBackends []*Backend
	totalWeight := 0
	for _, backend := range backends {
		if backend.IsHealthy() {
			// Find the weight setting that matches the ServiceName
			for _, weight := range lb.config.Weights {
				if weight.ServiceName == backend.ServiceName {
					backend.Weight = weight.Weight
					break
				}
			}
			// Use default 1 if weight is not set
			if backend.Weight == 0 {
				backend.Weight = 1
			}
			healthyBackends = append(healthyBackends, backend)
			totalWeight += backend.Weight
		}
	}

	if totalWeight == 0 {
		return nil
	}

	current := atomic.AddUint32(&state.current, 1)
	point := int(current) % totalWeight

	for _, backend := range healthyBackends {
		point -= backend.Weight
		if point < 0 {
			return backend
		}
	}

	return healthyBackends[0]
}

// Based on IP hash
func (lb *LoadBalancer) ipHashBackend(backends []*Backend, clientIP string) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// Converting IP to Hash Value
	hash := fnv.New32a()
	hash.Write([]byte(clientIP))
	hashValue := hash.Sum32()

	// Filtering only healthy backend
	var healthyBackends []*Backend
	for _, backend := range backends {
		if backend.IsHealthy() {
			healthyBackends = append(healthyBackends, backend)
		}
	}

	if len(healthyBackends) == 0 {
		return nil
	}

	return healthyBackends[hashValue%uint32(len(healthyBackends))]
}

// Random selection
func (lb *LoadBalancer) randomBackend(backends []*Backend) *Backend {
	if len(backends) == 0 {
		return nil
	}

	// Filtering only healthy backend
	var healthyBackends []*Backend
	for _, backend := range backends {
		if backend.IsHealthy() {
			healthyBackends = append(healthyBackends, backend)
		}
	}

	if len(healthyBackends) == 0 {
		return nil
	}

	return healthyBackends[rand.Intn(len(healthyBackends))]
}
