package controller

import (
	"testing"

	"github.com/somaz94/helios-lb/internal/metrics"
	"github.com/somaz94/helios-lb/internal/network"
)

func TestSplitRequestedIP(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		wantV4 string
		wantV6 string
	}{
		{"empty pins neither family", "", "", ""},
		{"IPv4 pins the v4 family", "192.0.2.10", "192.0.2.10", ""},
		{"IPv6 pins the v6 family", "2001:db8::10", "", "2001:db8::10"},
		{"surrounding space is trimmed", "  192.0.2.10  ", "192.0.2.10", ""},
		{"an unparseable value pins neither", "not-an-ip", "", ""},
		// An IPv4-mapped IPv6 literal is still an IPv4 address.
		{"IPv4-mapped IPv6 counts as v4", "::ffff:192.0.2.10", "::ffff:192.0.2.10", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v4, v6 := splitRequestedIP(tt.in)
			if v4 != tt.wantV4 || v6 != tt.wantV6 {
				t.Errorf("splitRequestedIP(%q) = (%q, %q), want (%q, %q)", tt.in, v4, v6, tt.wantV4, tt.wantV6)
			}
		})
	}
}

// With no request, allocation still draws from the pool.
func TestIPManager_AllocateIP_FallsBackToThePool(t *testing.T) {
	m := newTestIPManager()

	got, err := m.allocateIP("192.0.2.10-192.0.2.20", "")
	if err != nil {
		t.Fatalf("allocateIP() error = %v, want nil", err)
	}
	if got != "192.0.2.10" {
		t.Errorf("allocateIP() = %q, want the first pool address", got)
	}
}

// A requested address is handed back verbatim rather than the pool's next one.
func TestIPManager_AllocateIP_HonorsARequest(t *testing.T) {
	m := newTestIPManager()

	got, err := m.allocateIP("192.0.2.10-192.0.2.20", "192.0.2.17")
	if err != nil {
		t.Fatalf("allocateIP() error = %v, want nil", err)
	}
	if got != "192.0.2.17" {
		t.Errorf("allocateIP() = %q, want the requested address, not the pool's first", got)
	}
}

// The regression this path exists for: a taken request must fail loudly instead
// of silently yielding a different address.
func TestIPManager_AllocateIP_TakenRequestFails(t *testing.T) {
	m := newTestIPManager()
	const ipRange = "192.0.2.10-192.0.2.20"

	if _, err := m.allocateIP(ipRange, "192.0.2.17"); err != nil {
		t.Fatal(err)
	}

	got, err := m.allocateIP(ipRange, "192.0.2.17")
	if err == nil {
		t.Fatalf("allocateIP() = %q, want a failure for an address already taken", got)
	}
	if got != "" {
		t.Errorf("allocateIP() = %q, want no substitute address", got)
	}
}

// An unallocatable request is refused even though the address sits inside the
// CIDR block.
func TestIPManager_AllocateIP_RefusesBroadcastRequest(t *testing.T) {
	m := newTestIPManager()

	if got, err := m.allocateIP("192.0.2.0/24", "192.0.2.255"); err == nil {
		t.Errorf("allocateIP() = %q, want the broadcast address refused", got)
	}
}

// newTestIPManager builds an IPManager with a fresh allocator and no cluster.
func newTestIPManager() *IPManager {
	return &IPManager{
		NetworkMgr: network.NewNetworkManager(),
		Metrics:    metrics.NewMetricsRecorder(),
	}
}
