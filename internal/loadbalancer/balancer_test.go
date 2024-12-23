package loadbalancer

import (
	"testing"
	"time"
)

// Helper function to set test timeout
func runWithTimeout(t *testing.T, timeout time.Duration, test func(t *testing.T)) {
	completed := make(chan struct{})
	go func() {
		defer close(completed)
		test(t)
	}()

	select {
	case <-completed:
		return
	case <-time.After(timeout):
		t.Fatal("Test timed out")
	}
}

func createTestBackend(address string, serviceName string, weight int) *Backend {
	backend := &Backend{
		Address:     address,
		Port:        80,
		ServiceName: serviceName,
		Weight:      weight,
	}
	backend.SetHealthy(true)
	return backend
}

func TestLoadBalancer(t *testing.T) {
	tests := []struct {
		name     string
		method   BalancerType
		testFunc func(t *testing.T)
	}{
		{"RoundRobin", RoundRobin, testRoundRobin},
		{"LeastConnection", LeastConnection, testLeastConnection},
		{"WeightedRoundRobin", WeightedRoundRobin, testWeightedRoundRobin},
		{"IPHash", IPHash, testIPHash},
		{"RandomSelection", RandomSelection, testRandomSelection},
	}

	for _, tt := range tests {
		t.Run(string(tt.method), func(t *testing.T) {
			runWithTimeout(t, 500*time.Millisecond, tt.testFunc)
		})
	}
}

func testRoundRobin(t *testing.T) {
	balancer := NewLoadBalancer(BalancerConfig{Type: RoundRobin})
	defer balancer.Stop()

	backend1 := createTestBackend("192.168.1.1", "test-service", 1)
	backend2 := createTestBackend("192.168.1.2", "test-service", 1)

	balancer.AddBackend(backend1)
	balancer.AddBackend(backend2)

	got1 := balancer.NextBackend("test-service", "")
	got2 := balancer.NextBackend("test-service", "")
	got3 := balancer.NextBackend("test-service", "")

	if got1.Address == got2.Address {
		t.Error("Expected different backends in sequence")
	}
	if got1.Address != got3.Address {
		t.Error("Expected round robin to wrap around")
	}
}

func testLeastConnection(t *testing.T) {
	balancer := NewLoadBalancer(BalancerConfig{Type: LeastConnection})
	defer balancer.Stop()

	backend1 := createTestBackend("192.168.1.1", "test-service", 1)
	backend2 := createTestBackend("192.168.1.2", "test-service", 1)

	backend1.Connections = 5
	backend2.Connections = 2

	balancer.AddBackend(backend1)
	balancer.AddBackend(backend2)

	got := balancer.NextBackend("test-service", "")
	if got.Address != backend2.Address {
		t.Errorf("Expected backend with least connections (backend2), got %v", got)
	}
}

func testWeightedRoundRobin(t *testing.T) {
	balancer := NewLoadBalancer(BalancerConfig{Type: WeightedRoundRobin})
	defer balancer.Stop()

	backend1 := createTestBackend("192.168.1.1", "test-service", 2) // Weight: 2
	backend2 := createTestBackend("192.168.1.2", "test-service", 1) // Weight: 1

	balancer.AddBackend(backend1)
	balancer.AddBackend(backend2)

	selections := make(map[string]int)
	for i := 0; i < 30; i++ {
		backend := balancer.NextBackend("test-service", "")
		selections[backend.Address]++
	}

	// backend1 should be selected roughly twice as often as backend2
	ratio := float64(selections[backend1.Address]) / float64(selections[backend2.Address])
	if ratio < 1.5 || ratio > 2.5 {
		t.Errorf("Expected ratio close to 2.0, got %v", ratio)
	}
}

func testIPHash(t *testing.T) {
	balancer := NewLoadBalancer(BalancerConfig{Type: IPHash})
	defer balancer.Stop()

	backend1 := createTestBackend("192.168.1.1", "test-service", 1)
	backend2 := createTestBackend("192.168.1.2", "test-service", 1)

	balancer.AddBackend(backend1)
	balancer.AddBackend(backend2)

	clientIP1 := "10.0.0.1"
	clientIP2 := "10.0.0.2"

	// Same client IP should always get same backend
	first := balancer.NextBackend("test-service", clientIP1)
	for i := 0; i < 5; i++ {
		got := balancer.NextBackend("test-service", clientIP1)
		if got.Address != first.Address {
			t.Error("Expected same backend for same client IP")
		}
	}

	// Different client IP might get different backend
	different := balancer.NextBackend("test-service", clientIP2)
	if different.Address != first.Address {
		t.Log("Different client IP got different backend (expected behavior)")
	}
}

func testRandomSelection(t *testing.T) {
	balancer := NewLoadBalancer(BalancerConfig{Type: RandomSelection})
	defer balancer.Stop()

	backend1 := createTestBackend("192.168.1.1", "test-service", 1)
	backend2 := createTestBackend("192.168.1.2", "test-service", 1)

	balancer.AddBackend(backend1)
	balancer.AddBackend(backend2)

	selections := make(map[string]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		backend := balancer.NextBackend("test-service", "")
		selections[backend.Address]++
	}

	// Check that both backends were selected at least once
	if len(selections) != 2 {
		t.Error("Expected both backends to be selected")
	}

	// Check that distribution is roughly even (within 20% of 50/50)
	for _, count := range selections {
		ratio := float64(count) / float64(iterations)
		if ratio < 0.4 || ratio > 0.6 {
			t.Error("Random distribution appears to be biased")
		}
	}
}

func TestHealthCheck(t *testing.T) {
	balancer := NewLoadBalancer(BalancerConfig{
		Type:          RoundRobin,
		HealthCheck:   true,
		CheckInterval: time.Millisecond * 5,
	})

	backend := &Backend{
		Address:     "240.0.0.1", // non-existent address
		Port:        12345,
		ServiceName: "test-service",
	}
	backend.SetHealthy(true)
	balancer.AddBackend(backend)

	time.Sleep(time.Millisecond * 15)

	healthy := backend.IsHealthy()
	balancer.Stop()

	if healthy {
		t.Error("Expected backend to be marked as unhealthy")
	}
}
