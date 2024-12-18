package network

import (
	"testing"
)

func TestIPAllocator(t *testing.T) {
	t.Run("Valid IP range", func(t *testing.T) {
		allocator := NewIPAllocator()
		ip1, err := allocator.AllocateIP("192.168.1.1-192.168.1.3")
		if err != nil {
			t.Fatalf("Failed to allocate IP: %v", err)
		}
		if ip1 != "192.168.1.1" {
			t.Errorf("Expected first IP, got %s", ip1)
		}

		ip2, err := allocator.AllocateIP("192.168.1.1-192.168.1.3")
		if err != nil {
			t.Fatalf("Failed to allocate second IP: %v", err)
		}
		if ip2 != "192.168.1.2" {
			t.Errorf("Expected second IP, got %s", ip2)
		}
	})

	t.Run("Invalid IP range", func(t *testing.T) {
		allocator := NewIPAllocator()
		_, err := allocator.AllocateIP("invalid-range")
		if err == nil {
			t.Error("Expected error for invalid range")
		}
	})

	t.Run("IP release", func(t *testing.T) {
		allocator := NewIPAllocator()
		ip1, _ := allocator.AllocateIP("192.168.1.1-192.168.1.2")
		allocator.ReleaseIP(ip1)
		ip2, _ := allocator.AllocateIP("192.168.1.1-192.168.1.2")
		if ip1 != ip2 {
			t.Errorf("Expected released IP to be reallocated, got %s, want %s", ip2, ip1)
		}
	})

	t.Run("Single IP range", func(t *testing.T) {
		allocator := NewIPAllocator()
		ip1, err := allocator.AllocateIP("192.168.1.1-192.168.1.1")
		if err != nil {
			t.Fatalf("Failed to allocate IP: %v", err)
		}
		if ip1 != "192.168.1.1" {
			t.Errorf("Expected IP 192.168.1.1, got %s", ip1)
		}

		// 같은 IP를 다시 할당받을 수 있어야 함
		ip2, err := allocator.AllocateIP("192.168.1.1-192.168.1.1")
		if err != nil {
			t.Fatalf("Failed to reallocate same IP: %v", err)
		}
		if ip2 != "192.168.1.1" {
			t.Errorf("Expected same IP 192.168.1.1, got %s", ip2)
		}
	})

	t.Run("Multiple allocations in range", func(t *testing.T) {
		allocator := NewIPAllocator()
		ip1, _ := allocator.AllocateIP("192.168.1.1-192.168.1.2")
		ip2, _ := allocator.AllocateIP("192.168.1.1-192.168.1.2")
		_, err := allocator.AllocateIP("192.168.1.1-192.168.1.2")
		if err == nil {
			t.Error("Expected error when range is exhausted")
		}
		if ip1 == ip2 {
			t.Error("Expected different IPs to be allocated")
		}
	})
}
