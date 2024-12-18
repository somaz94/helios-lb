package loadbalancer

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/somaz94/helios-lb/internal/metrics"
)

type BalancerType string

const (
	RoundRobin      BalancerType = "roundrobin"
	LeastConnection BalancerType = "leastconnection"
)

// BalancerConfig defines the configuration for the load balancer
type BalancerConfig struct {
	Type           BalancerType
	HealthCheck    bool
	CheckInterval  time.Duration
	MetricsEnabled bool
}

// Backend represents a backend server
type Backend struct {
	Address     string
	Port        int
	healthy     int32
	Connections int32
	ServiceName string
}

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

type LoadBalancerStats struct {
	TotalConnections int32
	ActiveBackends   int
	HealthyBackends  int
}

// RoundRobinState stores the state for round-robin load balancing
type RoundRobinState struct {
	current uint32
}

type LoadBalancer struct {
	mu       sync.RWMutex
	backends map[string][]*Backend
	stats    map[string]*LoadBalancerStats
	rrStates map[string]*RoundRobinState
	config   BalancerConfig
	stopCh   chan struct{}
	wg       sync.WaitGroup
	checkWg  sync.WaitGroup
}

// NewLoadBalancer creates a new load balancer instance
func NewLoadBalancer(config BalancerConfig) *LoadBalancer {
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

func (lb *LoadBalancer) healthCheckLoop() {
	defer lb.wg.Done()

	ticker := time.NewTicker(lb.config.CheckInterval)
	defer ticker.Stop()

	// 초기 health check
	lb.checkWg.Add(1)
	go func() {
		lb.doHealthCheck()
		lb.checkWg.Done()
	}()

	for {
		select {
		case <-lb.stopCh:
			lb.checkWg.Wait()
			return
		case <-ticker.C:
			lb.checkWg.Add(1)
			go func() {
				lb.doHealthCheck()
				lb.checkWg.Done()
			}()
		}
	}
}

func (lb *LoadBalancer) doHealthCheck() {
	// RLock 사용 (Lock 대신)
	lb.mu.RLock()
	backends := make(map[string][]*Backend)
	// 백엔드 목록 복사
	for service, bkends := range lb.backends {
		backends[service] = append([]*Backend{}, bkends...)
	}
	lb.mu.RUnlock()

	// 락 없이 health check 수행
	for _, bkends := range backends {
		for _, backend := range bkends {
			d := net.Dialer{
				Timeout: time.Millisecond * 10,
			}
			conn, err := d.Dial("tcp", fmt.Sprintf("%s:%d", backend.Address, backend.Port))
			if err != nil {
				backend.SetHealthy(false)
				continue
			}
			if conn != nil {
				conn.Close()
				backend.SetHealthy(true)
			}
		}
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

// NextBackend returns the next backend server based on the balancing method
func (lb *LoadBalancer) NextBackend(serviceName string) *Backend {
	// 백엔드 목록과 상태를 한 번에 복사
	lb.mu.RLock()
	backends, exists := lb.backends[serviceName]
	if !exists || len(backends) == 0 {
		lb.mu.RUnlock()
		return nil
	}

	// 백엔드 목록 복사
	backendsCopy := make([]*Backend, len(backends))
	copy(backendsCopy, backends)
	lb.mu.RUnlock()

	switch lb.config.Type {
	case LeastConnection:
		return lb.leastConnectionBackend(backendsCopy)
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

	// rrStates에 대한 초기화 및 접근을 분리
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

// IncrementConnections increments the connection count for a backend
func (lb *LoadBalancer) IncrementConnections(backend *Backend) {
	atomic.AddInt32(&backend.Connections, 1)
}

// DecrementConnections decrements the connection count for a backend
func (lb *LoadBalancer) DecrementConnections(backend *Backend) {
	atomic.AddInt32(&backend.Connections, -1)
}

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
