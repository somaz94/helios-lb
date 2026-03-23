package v1

import (
	"context"
	"net"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

func TestValidateCreate(t *testing.T) {
	v := &HeliosConfigValidator{Client: nil}
	ctx := context.Background()

	t.Run("valid config", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec:       HeliosConfigSpec{IPRange: "10.0.0.1-10.0.0.10", Method: "RoundRobin"},
		}
		_, err := v.ValidateCreate(ctx, hc)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("invalid ipRange", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec:       HeliosConfigSpec{IPRange: "bad"},
		}
		_, err := v.ValidateCreate(ctx, hc)
		if err == nil {
			t.Error("expected error for invalid ipRange")
		}
	})

	t.Run("invalid port", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec: HeliosConfigSpec{
				IPRange: "10.0.0.1",
				Ports:   []PortConfig{{Port: 0}},
			},
		}
		_, err := v.ValidateCreate(ctx, hc)
		if err == nil {
			t.Error("expected error for invalid port")
		}
	})

	t.Run("invalid weights method mismatch", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec: HeliosConfigSpec{
				IPRange: "10.0.0.1",
				Method:  "RoundRobin",
				Weights: []WeightConfig{{ServiceName: "svc", Weight: 1}},
			},
		}
		_, err := v.ValidateCreate(ctx, hc)
		if err == nil {
			t.Error("expected error for weights with non-WRR method")
		}
	})

	t.Run("invalid health check", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec: HeliosConfigSpec{
				IPRange:     "10.0.0.1",
				HealthCheck: &HealthCheckConfig{Protocol: "HTTP"},
			},
		}
		_, err := v.ValidateCreate(ctx, hc)
		if err == nil {
			t.Error("expected error for HTTP without path")
		}
	})
}

func TestValidateUpdate(t *testing.T) {
	v := &HeliosConfigValidator{Client: nil}
	ctx := context.Background()

	old := &HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       HeliosConfigSpec{IPRange: "10.0.0.1"},
	}

	t.Run("valid update", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec:       HeliosConfigSpec{IPRange: "10.0.0.1-10.0.0.5"},
		}
		_, err := v.ValidateUpdate(ctx, old, hc)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("invalid update", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "test"},
			Spec:       HeliosConfigSpec{IPRange: "invalid"},
		}
		_, err := v.ValidateUpdate(ctx, old, hc)
		if err == nil {
			t.Error("expected error for invalid ipRange")
		}
	})
}

func TestValidateDelete(t *testing.T) {
	v := &HeliosConfigValidator{Client: nil}
	hc := &HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       HeliosConfigSpec{IPRange: "10.0.0.1"},
	}
	_, err := v.ValidateDelete(context.Background(), hc)
	if err != nil {
		t.Errorf("expected no error on delete, got %v", err)
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"single IP", "10.0.0.1", false},
		{"range", "10.0.0.1-10.0.0.5", false},
		{"CIDR", "10.0.0.0/24", false},
		{"IPv6 single", "fd00::1", false},
		{"IPv6 CIDR", "fd00::/120", false},
		{"invalid", "bad", true},
		{"invalid CIDR", "10.0.0.0/bad", true},
		{"invalid range IPs", "bad-worse", true},
		{"too many parts", "1-2-3", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, e, err := parseRange(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRange(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if err == nil && (s == nil || e == nil) {
				t.Errorf("parseRange(%q) returned nil IPs without error", tt.input)
			}
		})
	}
}

func TestDeepCopy(t *testing.T) {
	hc := &HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		Spec: HeliosConfigSpec{
			IPRange:  "10.0.0.1-10.0.0.10",
			Method:   "RoundRobin",
			Protocol: "TCP",
			Ports:    []PortConfig{{Port: 80, Protocol: "TCP"}, {Port: 443}},
			Weights:  []WeightConfig{{ServiceName: "svc1", Weight: 5}},
			NamespaceSelector: []string{"default", "production"},
			MaxAllocations:    10,
			HealthCheck: &HealthCheckConfig{
				Enabled:         true,
				IntervalSeconds: 5,
				TimeoutMs:       1000,
				Protocol:        "HTTP",
				HTTPPath:        "/healthz",
			},
		},
		Status: HeliosConfigStatus{
			AllocatedIPs: map[string]string{"svc1": "10.0.0.1", "svc2": "10.0.0.2"},
			State:        StateActive,
			Phase:        StateActive,
			Message:      "OK",
		},
	}

	copied := hc.DeepCopy()
	if copied.Name != hc.Name {
		t.Errorf("DeepCopy Name mismatch: got %s, want %s", copied.Name, hc.Name)
	}
	if copied.Spec.IPRange != hc.Spec.IPRange {
		t.Errorf("DeepCopy IPRange mismatch")
	}
	if len(copied.Spec.Ports) != len(hc.Spec.Ports) {
		t.Errorf("DeepCopy Ports length mismatch")
	}
	if len(copied.Spec.Weights) != len(hc.Spec.Weights) {
		t.Errorf("DeepCopy Weights length mismatch")
	}
	if copied.Spec.HealthCheck == nil || copied.Spec.HealthCheck.HTTPPath != "/healthz" {
		t.Errorf("DeepCopy HealthCheck mismatch")
	}
	if len(copied.Status.AllocatedIPs) != 2 {
		t.Errorf("DeepCopy AllocatedIPs length mismatch")
	}

	// Ensure deep copy is independent
	copied.Status.AllocatedIPs["svc3"] = "10.0.0.3"
	if len(hc.Status.AllocatedIPs) != 2 {
		t.Error("DeepCopy is not independent - modifying copy affected original")
	}

	// Test DeepCopyObject
	obj := hc.DeepCopyObject()
	if _, ok := obj.(*HeliosConfig); !ok {
		t.Error("DeepCopyObject did not return *HeliosConfig")
	}

	// Test HeliosConfigList DeepCopy
	list := &HeliosConfigList{Items: []HeliosConfig{*hc}}
	listCopy := list.DeepCopy()
	if len(listCopy.Items) != 1 {
		t.Errorf("HeliosConfigList DeepCopy Items length mismatch")
	}
	listObj := list.DeepCopyObject()
	if _, ok := listObj.(*HeliosConfigList); !ok {
		t.Error("HeliosConfigList DeepCopyObject did not return *HeliosConfigList")
	}

	// Test HeliosConfigSpec DeepCopy
	specCopy := hc.Spec.DeepCopy()
	if specCopy.IPRange != hc.Spec.IPRange {
		t.Error("Spec DeepCopy mismatch")
	}

	// Test HeliosConfigStatus DeepCopy
	statusCopy := hc.Status.DeepCopy()
	if len(statusCopy.AllocatedIPs) != 2 {
		t.Error("Status DeepCopy mismatch")
	}

	// Test PortConfig DeepCopy
	portCopy := hc.Spec.Ports[0].DeepCopy()
	if portCopy.Port != 80 {
		t.Error("PortConfig DeepCopy mismatch")
	}

	// Test WeightConfig DeepCopy
	weightCopy := hc.Spec.Weights[0].DeepCopy()
	if weightCopy.ServiceName != "svc1" {
		t.Error("WeightConfig DeepCopy mismatch")
	}

	// Test HealthCheckConfig DeepCopy
	hcCopy := hc.Spec.HealthCheck.DeepCopy()
	if hcCopy.Protocol != "HTTP" {
		t.Error("HealthCheckConfig DeepCopy mismatch")
	}

	// Test nil DeepCopy
	var nilHC *HeliosConfig
	if nilHC.DeepCopy() != nil {
		t.Error("nil HeliosConfig DeepCopy should return nil")
	}
	var nilList *HeliosConfigList
	if nilList.DeepCopy() != nil {
		t.Error("nil HeliosConfigList DeepCopy should return nil")
	}
	var nilSpec *HeliosConfigSpec
	if nilSpec.DeepCopy() != nil {
		t.Error("nil Spec DeepCopy should return nil")
	}
	var nilStatus *HeliosConfigStatus
	if nilStatus.DeepCopy() != nil {
		t.Error("nil Status DeepCopy should return nil")
	}
	var nilPort *PortConfig
	if nilPort.DeepCopy() != nil {
		t.Error("nil PortConfig DeepCopy should return nil")
	}
	var nilWeight *WeightConfig
	if nilWeight.DeepCopy() != nil {
		t.Error("nil WeightConfig DeepCopy should return nil")
	}
	var nilHealthCheck *HealthCheckConfig
	if nilHealthCheck.DeepCopy() != nil {
		t.Error("nil HealthCheckConfig DeepCopy should return nil")
	}
}

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = AddToScheme(s)
	return s
}

func TestCheckIPRangeOverlap(t *testing.T) {
	scheme := newTestScheme()

	existing := &HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "existing", Namespace: "default"},
		Spec:       HeliosConfigSpec{IPRange: "10.0.0.1-10.0.0.10"},
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	v := &HeliosConfigValidator{Client: cl}
	ctx := context.Background()

	t.Run("overlapping range rejected", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "new-config"},
			Spec:       HeliosConfigSpec{IPRange: "10.0.0.5-10.0.0.15"},
		}
		_, err := v.ValidateCreate(ctx, hc)
		if err == nil {
			t.Error("expected overlap error")
		}
	})

	t.Run("non-overlapping range accepted", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "new-config"},
			Spec:       HeliosConfigSpec{IPRange: "10.0.0.20-10.0.0.30"},
		}
		_, err := v.ValidateCreate(ctx, hc)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("update excludes self", func(t *testing.T) {
		hc := &HeliosConfig{
			ObjectMeta: metav1.ObjectMeta{Name: "existing"},
			Spec:       HeliosConfigSpec{IPRange: "10.0.0.1-10.0.0.10"},
		}
		_, err := v.ValidateUpdate(ctx, existing, hc)
		if err != nil {
			t.Errorf("expected no error for self-update, got %v", err)
		}
	})
}

func TestDeepCopy_EmptyStatus(t *testing.T) {
	// Test status with nil AllocatedIPs and nil Conditions
	hc := &HeliosConfig{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec:       HeliosConfigSpec{IPRange: "10.0.0.1"},
		Status:     HeliosConfigStatus{},
	}
	copied := hc.DeepCopy()
	if copied.Status.AllocatedIPs != nil {
		t.Error("expected nil AllocatedIPs in deep copy of empty status")
	}
}
