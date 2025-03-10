package loadbalancer

import (
	"sync/atomic"
	"time"
)

// NewLoadBalancer creates a new load balancer instance
func NewLoadBalancer(config BalancerConfig) *LoadBalancer {
	// Set default check interval if health check is enabled but interval is not positive
	if config.HealthCheck && config.CheckInterval <= 0 {
		config.CheckInterval = time.Second * 5 // 기본값 5초
	}

	lb := &LoadBalancer{
		backends: make(map[string][]*Backend),
		stats:    make(map[string]*LoadBalancerStats),
		rrStates: make(map[string]*RoundRobinState),
		config:   config,
		stopCh:   make(chan struct{}),
	}

	if config.HealthCheck {
		lb.wg.Add(1)
		go lb.healthCheckLoop()
	}

	return lb
}

// IncrementConnections increments the connection count for a backend
func (lb *LoadBalancer) IncrementConnections(backend *Backend) {
	atomic.AddInt32(&backend.Connections, 1)
}

// DecrementConnections decrements the connection count for a backend
func (lb *LoadBalancer) DecrementConnections(backend *Backend) {
	atomic.AddInt32(&backend.Connections, -1)
}

// Stop gracefully stops the load balancer
func (lb *LoadBalancer) Stop() {
	select {
	case <-lb.stopCh:
		return
	default:
		close(lb.stopCh)
	}
	lb.wg.Wait()
	lb.checkWg.Wait()
}
