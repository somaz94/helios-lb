package metrics

import (
	"testing"
)

func TestMetricsRecorder(t *testing.T) {
	recorder := NewMetricsRecorder()

	t.Run("Load balancer status metrics", func(t *testing.T) {
		// Test active status
		recorder.RecordLBStatus("lb1", "default", true)
		// Test inactive status
		recorder.RecordLBStatus("lb2", "test", false)
	})

	t.Run("Backend health metrics", func(t *testing.T) {
		// Test healthy backend
		recorder.RecordBackendHealth("192.168.1.1", "test-service", true)
		// Test unhealthy backend
		recorder.RecordBackendHealth("192.168.1.1", "test-service", false)
		// Test different service
		recorder.RecordBackendHealth("192.168.1.2", "other-service", true)
	})

	t.Run("Backend connections metrics", func(t *testing.T) {
		// Test zero connections
		recorder.RecordBackendConnections("192.168.1.1", "test-service", 0)
		// Test positive connections
		recorder.RecordBackendConnections("192.168.1.1", "test-service", 10)
		// Test negative connections (edge case)
		recorder.RecordBackendConnections("192.168.1.1", "test-service", -1)
		// Test different service
		recorder.RecordBackendConnections("192.168.1.2", "other-service", 5)
	})

	t.Run("Request duration metrics", func(t *testing.T) {
		// Test various durations
		recorder.RecordRequestDuration("test-service", 0.001)
		recorder.RecordRequestDuration("test-service", 0.1)
		recorder.RecordRequestDuration("test-service", 1.0)
		// Test different service
		recorder.RecordRequestDuration("other-service", 0.5)
	})

	t.Run("Operation metrics", func(t *testing.T) {
		// Test successful operations
		recorder.RecordOperation("test-service", "connect", "success")
		recorder.RecordOperation("test-service", "disconnect", "success")
		// Test failed operations
		recorder.RecordOperation("test-service", "connect", "failure")
		// Test different service
		recorder.RecordOperation("other-service", "connect", "success")
	})

	t.Run("IP allocation metrics", func(t *testing.T) {
		// Test IP allocation
		recorder.RecordIPAllocation("192.168.1.1", true)
		// Test IP deallocation
		recorder.RecordIPAllocation("192.168.1.1", false)
		// Test different IP
		recorder.RecordIPAllocation("192.168.1.2", true)
	})

	t.Run("Multiple services interaction", func(t *testing.T) {
		// Test multiple metrics for same service
		recorder.RecordBackendHealth("192.168.1.1", "service1", true)
		recorder.RecordBackendConnections("192.168.1.1", "service1", 5)
		recorder.RecordRequestDuration("service1", 0.05)
		recorder.RecordOperation("service1", "connect", "success")

		// Test multiple metrics for different service
		recorder.RecordBackendHealth("192.168.1.2", "service2", true)
		recorder.RecordBackendConnections("192.168.1.2", "service2", 10)
		recorder.RecordRequestDuration("service2", 0.08)
		recorder.RecordOperation("service2", "connect", "success")
	})

	t.Run("Edge cases", func(t *testing.T) {
		// Test empty service name
		recorder.RecordBackendHealth("192.168.1.1", "", true)
		recorder.RecordBackendConnections("192.168.1.1", "", 5)
		recorder.RecordRequestDuration("", 0.1)
		recorder.RecordOperation("", "connect", "success")

		// Test empty IP address
		recorder.RecordIPAllocation("", true)

		// Test extreme values
		recorder.RecordBackendConnections("192.168.1.1", "test", 1000000)
		recorder.RecordRequestDuration("test", 100.0)
	})
}
