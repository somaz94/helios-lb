package network

import (
	"net"
	"testing"
)

func TestCompareIPs(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{"identical IPv4", "192.0.2.10", "192.0.2.10", 0},
		{"IPv4 less than", "192.0.2.10", "192.0.2.11", -1},
		{"IPv4 greater than", "192.0.2.11", "192.0.2.10", 1},
		{"IPv4 across octets", "192.0.2.255", "192.0.3.0", -1},
		{"IPv4 first octet dominates", "10.0.0.1", "192.0.2.1", -1},
		{"identical IPv6", "2001:db8::1", "2001:db8::1", 0},
		{"IPv6 less than", "2001:db8::1", "2001:db8::2", -1},
		{"IPv6 greater than", "2001:db8::ff", "2001:db8::1", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareIPs(net.ParseIP(tt.a), net.ParseIP(tt.b))
			if got != tt.want {
				t.Errorf("CompareIPs(%s, %s) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// net.ParseIP hands back a 16-byte IPv4-mapped address, so comparing a parsed
// IPv4 against a 4-byte one only works because CompareIPs normalizes first.
// Without that, the two representations of the same address would differ.
func TestCompareIPs_NormalizesMixedRepresentations(t *testing.T) {
	parsed := net.ParseIP("192.0.2.10") // 16-byte IPv4-mapped form
	fourByte := parsed.To4()            // 4-byte form

	if len(parsed) == len(fourByte) {
		t.Skip("this Go version already returns a 4-byte IPv4 from ParseIP")
	}
	if got := CompareIPs(parsed, fourByte); got != 0 {
		t.Errorf("CompareIPs(16-byte, 4-byte) = %d, want 0 for the same address", got)
	}
}

func TestCompareIPs_OrdersAConsistentSequence(t *testing.T) {
	seq := []string{"192.0.2.1", "192.0.2.2", "192.0.2.10", "192.0.3.1", "203.0.113.1"}

	for i := 0; i < len(seq)-1; i++ {
		a, b := net.ParseIP(seq[i]), net.ParseIP(seq[i+1])
		if got := CompareIPs(a, b); got != -1 {
			t.Errorf("CompareIPs(%s, %s) = %d, want -1", seq[i], seq[i+1], got)
		}
		if got := CompareIPs(b, a); got != 1 {
			t.Errorf("CompareIPs(%s, %s) = %d, want 1", seq[i+1], seq[i], got)
		}
	}
}

func TestNetworkManager_MarkUsed(t *testing.T) {
	nm := NewNetworkManager()
	const ipRange = "192.0.2.10-192.0.2.12"

	nm.MarkUsed("192.0.2.10")
	nm.MarkUsed("192.0.2.11")

	got, err := nm.AllocateIP(ipRange)
	if err != nil {
		t.Fatalf("AllocateIP() error = %v, want nil", err)
	}
	if got != "192.0.2.12" {
		t.Errorf("AllocateIP() = %q, want the first address not marked used", got)
	}
}

// MarkUsed must reach the same allocator the manager allocates from, otherwise
// a reserved address could be handed out again.
func TestNetworkManager_MarkUsedExhaustsTheRange(t *testing.T) {
	nm := NewNetworkManager()
	const ipRange = "192.0.2.10-192.0.2.11"

	nm.MarkUsed("192.0.2.10")
	nm.MarkUsed("192.0.2.11")

	if got, err := nm.AllocateIP(ipRange); err == nil {
		t.Errorf("AllocateIP() = %q, want an error once every address is marked used", got)
	}
}

func TestNetworkManager_ReleaseIPMakesAnAddressAvailableAgain(t *testing.T) {
	nm := NewNetworkManager()
	const ipRange = "192.0.2.10-192.0.2.11"

	nm.MarkUsed("192.0.2.10")
	nm.MarkUsed("192.0.2.11")
	nm.ReleaseIP("192.0.2.11")

	got, err := nm.AllocateIP(ipRange)
	if err != nil {
		t.Fatalf("AllocateIP() error = %v, want nil after a release", err)
	}
	if got != "192.0.2.11" {
		t.Errorf("AllocateIP() = %q, want the released address", got)
	}
}
