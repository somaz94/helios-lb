package network

import (
	"errors"
	"strings"
	"testing"
)

// Addresses reused across the pair-of-addresses ranges below.
const (
	addrFirst  = "192.0.2.10"
	addrSecond = "192.0.2.11"
	pairRange  = "192.0.2.10-192.0.2.11"
)

func TestAllocateSpecificIP_HonorsTheRequest(t *testing.T) {
	a := NewIPAllocator()

	got, err := a.AllocateSpecificIP("192.0.2.10-192.0.2.20", "192.0.2.15")
	if err != nil {
		t.Fatalf("AllocateSpecificIP() error = %v, want nil", err)
	}
	if got != "192.0.2.15" {
		t.Errorf("AllocateSpecificIP() = %q, want the requested address", got)
	}
}

// The whole point of the specific path: a request is never quietly swapped for
// a different address.
func TestAllocateSpecificIP_NeverSubstitutes(t *testing.T) {
	a := NewIPAllocator()

	// Take the requested address first.
	if _, err := a.AllocateSpecificIP("192.0.2.10-192.0.2.20", "192.0.2.15"); err != nil {
		t.Fatal(err)
	}

	got, err := a.AllocateSpecificIP("192.0.2.10-192.0.2.20", "192.0.2.15")
	if err == nil {
		t.Fatalf("AllocateSpecificIP() = %q, want a failure rather than another address", got)
	}
	if !errors.Is(err, ErrIPUnavailable) {
		t.Errorf("error = %v, want it to wrap ErrIPUnavailable", err)
	}
	if got != "" {
		t.Errorf("AllocateSpecificIP() = %q, want no address alongside the error", got)
	}
}

// A previously allocated address becomes requestable again after release.
func TestAllocateSpecificIP_AfterRelease(t *testing.T) {
	a := NewIPAllocator()
	const ipRange = "192.0.2.10-192.0.2.20"

	if _, err := a.AllocateSpecificIP(ipRange, "192.0.2.15"); err != nil {
		t.Fatal(err)
	}
	a.ReleaseIP("192.0.2.15")

	got, err := a.AllocateSpecificIP(ipRange, "192.0.2.15")
	if err != nil {
		t.Fatalf("AllocateSpecificIP() error = %v, want nil after a release", err)
	}
	if got != "192.0.2.15" {
		t.Errorf("AllocateSpecificIP() = %q, want the released address back", got)
	}
}

func TestAllocateSpecificIP_Rejections(t *testing.T) {
	tests := []struct {
		name      string
		ipRange   string
		requested string
	}{
		{"outside the range", "192.0.2.10-192.0.2.20", "192.0.2.30"},
		{"below the range", "192.0.2.10-192.0.2.20", "192.0.2.1"},
		{"not an IP", "192.0.2.10-192.0.2.20", "not-an-ip"},
		{"empty", "192.0.2.10-192.0.2.20", ""},
		{"invalid range", "nonsense", "192.0.2.15"},
		// A CIDR's network and broadcast addresses are contained in the block but
		// are not addresses the allocator ever hands out.
		{"CIDR network address", "192.0.2.0/24", "192.0.2.0"},
		{"CIDR broadcast address", "192.0.2.0/24", "192.0.2.255"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewIPAllocator()
			got, err := a.AllocateSpecificIP(tt.ipRange, tt.requested)
			if err == nil {
				t.Fatalf("AllocateSpecificIP(%q, %q) = %q, want a rejection", tt.ipRange, tt.requested, got)
			}
			if !errors.Is(err, ErrIPUnavailable) {
				t.Errorf("error = %v, want it to wrap ErrIPUnavailable", err)
			}
		})
	}
}

// Requesting a specific address must not consume any other address, so a
// rejected request leaves the pool untouched.
func TestAllocateSpecificIP_RejectionLeavesThePoolIntact(t *testing.T) {
	a := NewIPAllocator()
	const ipRange = pairRange

	if _, err := a.AllocateSpecificIP(ipRange, "192.0.2.99"); err == nil {
		t.Fatal("expected the out-of-range request to fail")
	}

	// Both addresses must still be available.
	for _, want := range []string{addrFirst, addrSecond} {
		got, err := a.AllocateIP(ipRange)
		if err != nil {
			t.Fatalf("AllocateIP() error = %v, want the pool untouched", err)
		}
		if got != want {
			t.Errorf("AllocateIP() = %q, want %q", got, want)
		}
	}
}

// A specifically allocated address must be excluded from later pool draws.
func TestAllocateSpecificIP_RemovesTheAddressFromThePool(t *testing.T) {
	a := NewIPAllocator()
	const ipRange = pairRange

	if _, err := a.AllocateSpecificIP(ipRange, addrFirst); err != nil {
		t.Fatal(err)
	}

	got, err := a.AllocateIP(ipRange)
	if err != nil {
		t.Fatalf("AllocateIP() error = %v, want nil", err)
	}
	if got != addrSecond {
		t.Errorf("AllocateIP() = %q, want the address not already taken", got)
	}
}

func TestAllocateSpecificIP_IPv6(t *testing.T) {
	a := NewIPAllocator()

	got, err := a.AllocateSpecificIP("2001:db8::10-2001:db8::20", "2001:db8::15")
	if err != nil {
		t.Fatalf("AllocateSpecificIP() error = %v, want nil", err)
	}
	if got != "2001:db8::15" {
		t.Errorf("AllocateSpecificIP() = %q, want the requested address", got)
	}
}

// IPAllocatable is stricter than IPInRange, and the difference is exactly the
// addresses AllocateIP refuses to hand out.
func TestIPAllocatable_VersusIPInRange(t *testing.T) {
	const cidr = "192.0.2.0/24"

	tests := []struct {
		ip            string
		wantInRange   bool
		wantAllocable bool
	}{
		{"192.0.2.0", true, false},   // network address
		{"192.0.2.255", true, false}, // broadcast address
		{"192.0.2.1", true, true},
		{"192.0.2.254", true, true},
		{"192.0.3.1", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			if got := IPInRange(tt.ip, cidr); got != tt.wantInRange {
				t.Errorf("IPInRange(%s) = %v, want %v", tt.ip, got, tt.wantInRange)
			}
			if got := IPAllocatable(tt.ip, cidr); got != tt.wantAllocable {
				t.Errorf("IPAllocatable(%s) = %v, want %v", tt.ip, got, tt.wantAllocable)
			}
		})
	}
}

// The first address AllocateIP yields must itself be allocatable, which keeps
// the predicate and the allocator from drifting apart.
func TestIPAllocatable_AgreesWithAllocateIP(t *testing.T) {
	for _, ipRange := range []string{
		"192.0.2.0/24",
		"192.0.2.0/30",
		"192.0.2.10-192.0.2.20",
		"192.0.2.10",
		"2001:db8::/120",
	} {
		t.Run(ipRange, func(t *testing.T) {
			got, err := NewIPAllocator().AllocateIP(ipRange)
			if err != nil {
				t.Fatalf("AllocateIP(%q) error = %v", ipRange, err)
			}
			if !IPAllocatable(got, ipRange) {
				t.Errorf("AllocateIP yielded %s but IPAllocatable(%s, %s) = false", got, got, ipRange)
			}
		})
	}
}

func TestErrIPUnavailable_MessageNamesTheAddress(t *testing.T) {
	a := NewIPAllocator()

	_, err := a.AllocateSpecificIP("192.0.2.10-192.0.2.20", "192.0.2.99")
	if err == nil {
		t.Fatal("expected a rejection")
	}
	if !strings.Contains(err.Error(), "192.0.2.99") {
		t.Errorf("error = %q, want it to name the requested address", err.Error())
	}
}

func TestIPAllocatable_InvalidInputs(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		ipRange string
	}{
		{"unparseable IP", "not-an-ip", "192.0.2.0/24"},
		{"empty IP", "", "192.0.2.0/24"},
		{"unparseable range", "192.0.2.1", "nonsense"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if IPAllocatable(tt.ip, tt.ipRange) {
				t.Errorf("IPAllocatable(%q, %q) = true, want false", tt.ip, tt.ipRange)
			}
		})
	}
}

// The manager must reach the same allocator it allocates the pool from, so a
// specifically requested address is excluded from later pool draws.
func TestNetworkManager_AllocateSpecificIP(t *testing.T) {
	nm := NewNetworkManager()
	const ipRange = pairRange

	got, err := nm.AllocateSpecificIP(ipRange, addrFirst)
	if err != nil {
		t.Fatalf("AllocateSpecificIP() error = %v, want nil", err)
	}
	if got != addrFirst {
		t.Errorf("AllocateSpecificIP() = %q, want the requested address", got)
	}

	next, err := nm.AllocateIP(ipRange)
	if err != nil {
		t.Fatalf("AllocateIP() error = %v, want nil", err)
	}
	if next != addrSecond {
		t.Errorf("AllocateIP() = %q, want the address not already requested", next)
	}
}

func TestNetworkManager_AllocateSpecificIP_Rejects(t *testing.T) {
	nm := NewNetworkManager()

	if got, err := nm.AllocateSpecificIP(pairRange, "192.0.2.99"); err == nil {
		t.Errorf("AllocateSpecificIP() = %q, want an out-of-range request refused", got)
	}
}
