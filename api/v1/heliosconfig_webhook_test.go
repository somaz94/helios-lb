package v1

import (
	"net"
	"testing"
)

func TestValidateIPRange(t *testing.T) {
	tests := []struct {
		name    string
		ipRange string
		wantErr bool
	}{
		{"valid single IPv4", "192.168.1.1", false},
		{"valid IPv4 range", "192.168.1.1-192.168.1.10", false},
		{"valid IPv4 CIDR", "192.168.1.0/24", false},
		{"valid single IPv6", "fd00::1", false},
		{"valid IPv6 range", "fd00::1-fd00::ff", false},
		{"valid IPv6 CIDR", "fd00::/120", false},
		{"empty range", "", true},
		{"invalid format", "not-an-ip", true},
		{"invalid CIDR", "192.168.1.0/abc", true},
		{"start greater than end", "192.168.1.10-192.168.1.1", true},
		{"mixed IPv4/IPv6 range", "192.168.1.1-fd00::1", true},
		{"too many hyphens", "1.2.3.4-5.6.7.8-9.10.11.12", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIPRange(tt.ipRange)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateIPRange(%q) error = %v, wantErr %v", tt.ipRange, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePorts(t *testing.T) {
	tests := []struct {
		name    string
		ports   []PortConfig
		wantErr bool
	}{
		{"valid ports", []PortConfig{{Port: 80}, {Port: 443}}, false},
		{"empty ports", []PortConfig{}, false},
		{"port with protocol", []PortConfig{{Port: 80, Protocol: "TCP"}}, false},
		{"port zero", []PortConfig{{Port: 0}}, true},
		{"port too high", []PortConfig{{Port: 70000}}, true},
		{"invalid protocol", []PortConfig{{Port: 80, Protocol: "SCTP"}}, true},
		{"duplicate ports", []PortConfig{{Port: 80}, {Port: 80}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePorts(tt.ports)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePorts() error = %v, wantErr %v", tt.ports, tt.wantErr)
			}
		})
	}
}

func TestValidateWeights(t *testing.T) {
	tests := []struct {
		name    string
		weights []WeightConfig
		method  string
		wantErr bool
	}{
		{"no weights", nil, "RoundRobin", false},
		{"valid weights", []WeightConfig{{ServiceName: "svc1", Weight: 5}}, "WeightedRoundRobin", false},
		{"weights with wrong method", []WeightConfig{{ServiceName: "svc1", Weight: 5}}, "RoundRobin", true},
		{"weight too low", []WeightConfig{{ServiceName: "svc1", Weight: 0}}, "WeightedRoundRobin", true},
		{"weight too high", []WeightConfig{{ServiceName: "svc1", Weight: 101}}, "WeightedRoundRobin", true},
		{"empty service name", []WeightConfig{{ServiceName: "", Weight: 5}}, "WeightedRoundRobin", true},
		{"duplicate service", []WeightConfig{
			{ServiceName: "svc1", Weight: 5},
			{ServiceName: "svc1", Weight: 10},
		}, "WeightedRoundRobin", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWeights(tt.weights, tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWeights() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateHealthCheck(t *testing.T) {
	tests := []struct {
		name    string
		hc      *HealthCheckConfig
		wantErr bool
	}{
		{"nil config", nil, false},
		{"valid TCP", &HealthCheckConfig{Protocol: "TCP"}, false},
		{"valid HTTP", &HealthCheckConfig{Protocol: "HTTP", HTTPPath: "/health"}, false},
		{"invalid protocol", &HealthCheckConfig{Protocol: "GRPC"}, true},
		{"HTTP without path", &HealthCheckConfig{Protocol: "HTTP"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHealthCheck(tt.hc)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateHealthCheck() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCompareIPs(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"equal IPv4", "192.168.1.1", "192.168.1.1", 0},
		{"a < b IPv4", "192.168.1.1", "192.168.1.2", -1},
		{"a > b IPv4", "192.168.1.2", "192.168.1.1", 1},
		{"equal IPv6", "fd00::1", "fd00::1", 0},
		{"a < b IPv6", "fd00::1", "fd00::2", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := normalizeIPForValidation(parseTestIP(t, tt.a))
			b := normalizeIPForValidation(parseTestIP(t, tt.b))
			got := compareIPs(a, b)
			if got != tt.want {
				t.Errorf("compareIPs(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func parseTestIP(t *testing.T, s string) []byte {
	t.Helper()
	ip := net.ParseIP(s)
	if ip == nil {
		t.Fatalf("invalid test IP: %s", s)
	}
	return ip
}
