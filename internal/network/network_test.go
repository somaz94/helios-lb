package network

import (
	"testing"
)

func TestNetworkManager(t *testing.T) {
	t.Run("Create NetworkManager", func(t *testing.T) {
		nm := NewNetworkManager()
		if nm == nil {
			t.Fatal("Expected non-nil NetworkManager")
		}
		if nm.ipAllocator == nil {
			t.Fatal("Expected non-nil IPAllocator in NetworkManager")
		}
	})

	t.Run("Allocate and Release IP", func(t *testing.T) {
		nm := NewNetworkManager()
		ipRange := "192.168.1.1-192.168.1.10"

		// Test IP allocation
		ip1, err := nm.AllocateIP(ipRange)
		if err != nil {
			t.Fatalf("Failed to allocate IP: %v", err)
		}
		if ip1 != "192.168.1.1" {
			t.Errorf("Expected first IP to be 192.168.1.1, got %s", ip1)
		}

		// Release and reallocate the same IP
		nm.ReleaseIP(ip1)
		ip2, err := nm.AllocateIP(ipRange)
		if err != nil {
			t.Fatalf("Failed to reallocate IP: %v", err)
		}
		if ip2 != "192.168.1.1" {
			t.Errorf("Expected to get released IP %s back, got %s", ip1, ip2)
		}
	})

	t.Run("Invalid IP Range", func(t *testing.T) {
		nm := NewNetworkManager()
		_, err := nm.AllocateIP("invalid-range")
		if err == nil {
			t.Error("Expected error for invalid IP range")
		}
	})

	t.Run("Single IP Allocation", func(t *testing.T) {
		nm := NewNetworkManager()
		singleIP := "192.168.1.1"

		// First allocation
		ip1, err := nm.AllocateIP(singleIP)
		if err != nil {
			t.Fatalf("Failed to allocate single IP: %v", err)
		}
		if ip1 != singleIP {
			t.Errorf("Expected IP %s, got %s", singleIP, ip1)
		}

		// Second allocation of same IP should succeed
		ip2, err := nm.AllocateIP(singleIP)
		if err != nil {
			t.Fatalf("Failed to allocate same IP again: %v", err)
		}
		if ip2 != singleIP {
			t.Errorf("Expected same IP %s, got %s", singleIP, ip2)
		}
	})

	t.Run("Multiple IP Range Allocations", func(t *testing.T) {
		nm := NewNetworkManager()
		ipRange1 := "192.168.1.1-192.168.1.2"
		ipRange2 := "192.168.2.1-192.168.2.2"

		// Allocate from first range
		ip1, err := nm.AllocateIP(ipRange1)
		if err != nil {
			t.Fatalf("Failed to allocate from first range: %v", err)
		}

		// Allocate from second range
		ip2, err := nm.AllocateIP(ipRange2)
		if err != nil {
			t.Fatalf("Failed to allocate from second range: %v", err)
		}

		if ip1 == ip2 {
			t.Error("Expected different IPs from different ranges")
		}
	})
}
