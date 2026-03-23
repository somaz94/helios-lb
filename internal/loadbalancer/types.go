package loadbalancer

import (
	"sync"
	"time"
)

type BalancerType string

const (
	RoundRobin         BalancerType = "roundrobin"
	LeastConnection    BalancerType = "leastconnection"
	WeightedRoundRobin BalancerType = "weightedroundrobin"
	IPHash             BalancerType = "iphash"
	RandomSelection    BalancerType = "random"
)

// Algorithm defines the interface for load balancing algorithms.
// Implementations must be safe for concurrent use.
type Algorithm interface {
	// Select picks a backend from the given list based on the algorithm's strategy.
	// clientIP is used by IP-hash based algorithms and may be empty for others.
	Select(backends []*Backend, serviceName string, clientIP string) *Backend
}

type Weight struct {
	ServiceName string `json:"serviceName"`
	Weight      int    `json:"weight"`
}

type BalancerConfig struct {
	Type            BalancerType
	HealthCheck     bool
	CheckInterval   time.Duration
	MetricsEnabled  bool
	Weights         []Weight
	HealthCheckOpts HealthCheckOptions
}

type Backend struct {
	Address     string
	Port        int
	healthy     int32
	Connections int32
	ServiceName string
	Weight      int
}

type LoadBalancerStats struct {
	TotalConnections int32
	ActiveBackends   int
	HealthyBackends  int
}

type RoundRobinState struct {
	current uint32
}

type LoadBalancer struct {
	mu        sync.RWMutex
	backends  map[string][]*Backend
	stats     map[string]*LoadBalancerStats
	rrStates  map[string]*RoundRobinState
	config    BalancerConfig
	algorithm Algorithm
	stopCh    chan struct{}
	wg        sync.WaitGroup
	checkWg   sync.WaitGroup
}
