package metrics

import (
	"testing"
)

func TestMetricsRecorder(t *testing.T) {
	recorder := NewMetricsRecorder()

	t.Run("Backend health metrics", func(t *testing.T) {
		recorder.RecordBackendHealth("192.168.1.1", "test-service", true)
		recorder.RecordBackendHealth("192.168.1.1", "test-service", false)
	})

	t.Run("Backend connections metrics", func(t *testing.T) {
		recorder.RecordBackendConnections("192.168.1.1", "test-service", 0)
		recorder.RecordBackendConnections("192.168.1.1", "test-service", 10)
		recorder.RecordBackendConnections("192.168.1.1", "test-service", -1)
	})

	t.Run("Multiple services", func(t *testing.T) {
		recorder.RecordBackendHealth("192.168.1.1", "service1", true)
		recorder.RecordBackendHealth("192.168.1.2", "service2", true)
		recorder.RecordBackendConnections("192.168.1.1", "service1", 5)
		recorder.RecordBackendConnections("192.168.1.2", "service2", 10)
	})
}
