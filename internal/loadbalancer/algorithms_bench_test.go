package loadbalancer

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// benchServiceName is the service the benchmarks select backends for.
const benchServiceName = "bench-svc"

func newBenchBackends(n int) []*Backend {
	backends := make([]*Backend, n)
	for i := range backends {
		b := &Backend{
			Address:     fmt.Sprintf("10.0.0.%d", i+1),
			Port:        8080,
			ServiceName: benchServiceName,
			Weight:      1,
		}
		atomic.StoreInt32(&b.healthy, 1)
		backends[i] = b
	}
	return backends
}

func BenchmarkRoundRobin(b *testing.B) {
	for _, n := range []int{3, 10, 100} {
		b.Run(fmt.Sprintf("backends-%d", n), func(b *testing.B) {
			algo := &roundRobinAlgorithm{}
			backends := newBenchBackends(n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				algo.Select(backends, benchServiceName, "")
			}
		})
	}
}

func BenchmarkLeastConnection(b *testing.B) {
	for _, n := range []int{3, 10, 100} {
		b.Run(fmt.Sprintf("backends-%d", n), func(b *testing.B) {
			algo := &leastConnectionAlgorithm{}
			backends := newBenchBackends(n)
			// Simulate varying connection counts
			for i, backend := range backends {
				atomic.StoreInt32(&backend.Connections, int32(i%5))
			}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				algo.Select(backends, benchServiceName, "")
			}
		})
	}
}

func BenchmarkWeightedRoundRobin(b *testing.B) {
	for _, n := range []int{3, 10, 100} {
		b.Run(fmt.Sprintf("backends-%d", n), func(b *testing.B) {
			weights := make([]Weight, n)
			for i := range weights {
				weights[i] = Weight{
					ServiceName: benchServiceName,
					Weight:      (i%5 + 1),
				}
			}
			algo := &weightedRoundRobinAlgorithm{weights: weights}
			backends := newBenchBackends(n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				algo.Select(backends, benchServiceName, "")
			}
		})
	}
}

func BenchmarkIPHash(b *testing.B) {
	for _, n := range []int{3, 10, 100} {
		b.Run(fmt.Sprintf("backends-%d", n), func(b *testing.B) {
			algo := &ipHashAlgorithm{}
			backends := newBenchBackends(n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				algo.Select(backends, benchServiceName, fmt.Sprintf("192.168.1.%d", i%256))
			}
		})
	}
}

func BenchmarkRandom(b *testing.B) {
	for _, n := range []int{3, 10, 100} {
		b.Run(fmt.Sprintf("backends-%d", n), func(b *testing.B) {
			algo := &randomAlgorithm{}
			backends := newBenchBackends(n)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				algo.Select(backends, benchServiceName, "")
			}
		})
	}
}

// BenchmarkAllAlgorithms_Parallel tests concurrent access performance
func BenchmarkAllAlgorithms_Parallel(b *testing.B) {
	backends := newBenchBackends(10)

	algorithms := map[string]Algorithm{
		"RoundRobin":         &roundRobinAlgorithm{},
		"LeastConnection":    &leastConnectionAlgorithm{},
		"WeightedRoundRobin": &weightedRoundRobinAlgorithm{weights: []Weight{{ServiceName: benchServiceName, Weight: 1}}},
		"IPHash":             &ipHashAlgorithm{},
		"Random":             &randomAlgorithm{},
	}

	for name, algo := range algorithms {
		b.Run(name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				i := 0
				for pb.Next() {
					algo.Select(backends, benchServiceName, fmt.Sprintf("10.0.0.%d", i%256))
					i++
				}
			})
		})
	}
}
