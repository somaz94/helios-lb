package loadbalancer

import (
	"testing"
	"time"
)

// 테스트 타임아웃을 설정하는 헬퍼 함수
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

func TestLoadBalancer(t *testing.T) {
	t.Run("Empty balancer", func(t *testing.T) {
		balancer := NewLoadBalancer(BalancerConfig{
			Type:        RoundRobin,
			HealthCheck: false,
		})
		defer balancer.Stop()

		if backend := balancer.NextBackend("test-service"); backend != nil {
			t.Error("Expected nil backend from empty balancer")
		}
	})

	t.Run("Single backend", func(t *testing.T) {
		runWithTimeout(t, 500*time.Millisecond, func(t *testing.T) {
			balancer := NewLoadBalancer(BalancerConfig{
				Type:        RoundRobin,
				HealthCheck: false,
			})
			defer balancer.Stop()

			backend1 := &Backend{
				Address:     "192.168.1.1",
				Port:        80,
				ServiceName: "test-service",
			}
			backend1.SetHealthy(true)
			balancer.AddBackend(backend1)

			for i := 0; i < 3; i++ {
				if got := balancer.NextBackend("test-service"); got.Address != backend1.Address {
					t.Errorf("Expected backend1, got %v", got)
				}
			}
		})
	})

	t.Run("Multiple backends round-robin", func(t *testing.T) {
		runWithTimeout(t, 500*time.Millisecond, func(t *testing.T) {
			balancer := NewLoadBalancer(BalancerConfig{
				Type:        RoundRobin,
				HealthCheck: false,
			})
			defer balancer.Stop()

			backend1 := &Backend{
				Address:     "192.168.1.1",
				Port:        80,
				ServiceName: "test-service",
			}
			backend1.SetHealthy(true)

			backend2 := &Backend{
				Address:     "192.168.1.2",
				Port:        80,
				ServiceName: "test-service",
			}
			backend2.SetHealthy(true)

			balancer.AddBackend(backend1)
			balancer.AddBackend(backend2)

			got1 := balancer.NextBackend("test-service")
			got2 := balancer.NextBackend("test-service")
			got3 := balancer.NextBackend("test-service")

			if got1.Address == got2.Address {
				t.Error("Expected different backends in sequence")
			}
			if got1.Address != got3.Address {
				t.Error("Expected round robin to wrap around")
			}
		})
	})

	t.Run("Multiple backends least-conn", func(t *testing.T) {
		runWithTimeout(t, 100*time.Millisecond, func(t *testing.T) {
			balancer := NewLoadBalancer(BalancerConfig{
				Type:        LeastConnection,
				HealthCheck: false,
			})
			defer balancer.Stop()

			backend1 := &Backend{
				Address:     "192.168.1.1",
				Port:        80,
				ServiceName: "test-service",
				Connections: 5,
			}
			backend1.SetHealthy(true)

			backend2 := &Backend{
				Address:     "192.168.1.2",
				Port:        80,
				ServiceName: "test-service",
				Connections: 2,
			}
			backend2.SetHealthy(true)

			balancer.AddBackend(backend1)
			balancer.AddBackend(backend2)

			got := balancer.NextBackend("test-service")
			if got.Address != backend2.Address {
				t.Errorf("Expected backend with least connections (backend2), got %v", got)
			}
		})
	})
}

// health check 테스트를 별도로 분리
func TestHealthCheck(t *testing.T) {
	balancer := NewLoadBalancer(BalancerConfig{
		Type:          RoundRobin,
		HealthCheck:   true,
		CheckInterval: time.Millisecond * 5,
	})

	backend := &Backend{
		Address:     "240.0.0.1",
		Port:        12345,
		ServiceName: "test-service",
	}
	backend.SetHealthy(true)
	balancer.AddBackend(backend)

	// 더 짧은 대기 시간
	time.Sleep(time.Millisecond * 15)

	healthy := backend.IsHealthy()
	balancer.Stop()

	if healthy {
		t.Error("Expected backend to be marked as unhealthy")
	}
}
