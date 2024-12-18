package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// 로드밸런서 상태 메트릭
	lbStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helios_lb_status",
			Help: "Status of the load balancer (1 for active, 0 for inactive)",
		},
		[]string{"name", "namespace"},
	)

	// 백엔드 연결 수 메트릭
	backendConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helios_backend_connections",
			Help: "Number of active connections per backend",
		},
		[]string{"backend_address", "service_name"},
	)

	// 백엔드 상태 메트릭
	backendHealth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helios_backend_health",
			Help: "Health status of backend (1 for healthy, 0 for unhealthy)",
		},
		[]string{"backend_address", "service_name"},
	)

	// 로드밸런서 요청 처리 시간
	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "helios_request_duration_seconds",
			Help:    "Time taken to process requests",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"service_name"},
	)

	// 로드밸런싱 작업 성공/실패 카운터
	operationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "helios_operations_total",
			Help: "Total number of load balancing operations",
		},
		[]string{"service_name", "operation", "status"},
	)

	// IP 할당 메트릭
	ipAllocationStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "helios_ip_allocation_status",
			Help: "Status of IP allocation (1 for allocated, 0 for free)",
		},
		[]string{"ip_address"},
	)
)

func init() {
	// Register metrics with the global prometheus registry
	metrics.Registry.MustRegister(
		lbStatus,
		backendConnections,
		backendHealth,
		requestDuration,
		operationTotal,
		ipAllocationStatus,
	)
}

// MetricsRecorder provides methods to record metrics
type MetricsRecorder struct{}

func NewMetricsRecorder() *MetricsRecorder {
	return &MetricsRecorder{}
}

// RecordLBStatus records the load balancer status
func (m *MetricsRecorder) RecordLBStatus(name, namespace string, active bool) {
	value := 0.0
	if active {
		value = 1.0
	}
	lbStatus.WithLabelValues(name, namespace).Set(value)
}

// RecordBackendConnections records the number of connections to a backend
func (m *MetricsRecorder) RecordBackendConnections(backendAddr, serviceName string, connections float64) {
	backendConnections.WithLabelValues(backendAddr, serviceName).Set(connections)
}

// RecordBackendHealth records the health status of a backend
func (m *MetricsRecorder) RecordBackendHealth(backendAddr, serviceName string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	backendHealth.WithLabelValues(backendAddr, serviceName).Set(value)
}

// RecordRequestDuration records the duration of a request
func (m *MetricsRecorder) RecordRequestDuration(serviceName string, duration float64) {
	requestDuration.WithLabelValues(serviceName).Observe(duration)
}

// RecordOperation records a load balancing operation
func (m *MetricsRecorder) RecordOperation(serviceName, operation, status string) {
	operationTotal.WithLabelValues(serviceName, operation, status).Inc()
}

// RecordIPAllocation records IP allocation status
func (m *MetricsRecorder) RecordIPAllocation(ipAddress string, allocated bool) {
	value := 0.0
	if allocated {
		value = 1.0
	}
	ipAllocationStatus.WithLabelValues(ipAddress).Set(value)
}
