package loadbalancer

import (
	"hash/fnv"
	"math/rand"
	"sync"
	"sync/atomic"
)

// Compile-time interface checks
var (
	_ Algorithm = (*roundRobinAlgorithm)(nil)
	_ Algorithm = (*leastConnectionAlgorithm)(nil)
	_ Algorithm = (*weightedRoundRobinAlgorithm)(nil)
	_ Algorithm = (*ipHashAlgorithm)(nil)
	_ Algorithm = (*randomAlgorithm)(nil)
)

// NewAlgorithm creates an Algorithm implementation for the given type.
func NewAlgorithm(balancerType BalancerType, weights []Weight) Algorithm {
	switch balancerType {
	case LeastConnection:
		return &leastConnectionAlgorithm{}
	case WeightedRoundRobin:
		return &weightedRoundRobinAlgorithm{weights: weights}
	case IPHash:
		return &ipHashAlgorithm{}
	case RandomSelection:
		return &randomAlgorithm{}
	default:
		return &roundRobinAlgorithm{}
	}
}

// NextBackend returns the next backend server using the configured algorithm.
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

	return lb.algorithm.Select(backendsCopy, serviceName, clientIP)
}

// --- RoundRobin ---

type roundRobinAlgorithm struct {
	mu     sync.Mutex
	states map[string]*RoundRobinState
}

func (a *roundRobinAlgorithm) Select(backends []*Backend, serviceName string, _ string) *Backend {
	if len(backends) == 0 {
		return nil
	}

	a.mu.Lock()
	if a.states == nil {
		a.states = make(map[string]*RoundRobinState)
	}
	if _, exists := a.states[serviceName]; !exists {
		a.states[serviceName] = &RoundRobinState{}
	}
	state := a.states[serviceName]
	a.mu.Unlock()

	index := atomic.AddUint32(&state.current, 1) % uint32(len(backends))
	return backends[index]
}

// --- LeastConnection ---

type leastConnectionAlgorithm struct{}

func (a *leastConnectionAlgorithm) Select(backends []*Backend, _ string, _ string) *Backend {
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

// --- WeightedRoundRobin ---

type weightedRoundRobinAlgorithm struct {
	mu      sync.Mutex
	states  map[string]*RoundRobinState
	weights []Weight
}

func (a *weightedRoundRobinAlgorithm) Select(backends []*Backend, serviceName string, _ string) *Backend {
	if len(backends) == 0 {
		return nil
	}

	a.mu.Lock()
	if a.states == nil {
		a.states = make(map[string]*RoundRobinState)
	}
	if _, exists := a.states[serviceName]; !exists {
		a.states[serviceName] = &RoundRobinState{}
	}
	state := a.states[serviceName]
	a.mu.Unlock()

	var healthyBackends []*Backend
	totalWeight := 0
	for _, backend := range backends {
		if backend.IsHealthy() {
			for _, weight := range a.weights {
				if weight.ServiceName == backend.ServiceName {
					backend.Weight = weight.Weight
					break
				}
			}
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

// --- IPHash ---

type ipHashAlgorithm struct{}

func (a *ipHashAlgorithm) Select(backends []*Backend, _ string, clientIP string) *Backend {
	if len(backends) == 0 {
		return nil
	}

	hash := fnv.New32a()
	hash.Write([]byte(clientIP))
	hashValue := hash.Sum32()

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

// --- Random ---

type randomAlgorithm struct{}

func (a *randomAlgorithm) Select(backends []*Backend, _ string, _ string) *Backend {
	if len(backends) == 0 {
		return nil
	}

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
