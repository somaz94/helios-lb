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

	t.Run("Invalid format with multiple hyphens", func(t *testing.T) {
		allocator := NewIPAllocator()
		_, err := allocator.AllocateIP("192.168.1.1-192.168.1.2-192.168.1.3")
		if err == nil {
			t.Error("Expected error for range with multiple hyphens")
		}
	})
}

func TestIPInRange(t *testing.T) {
	t.Run("IP within range", func(t *testing.T) {
		if !IPInRange("192.168.1.5", "192.168.1.1-192.168.1.10") {
			t.Error("Expected 192.168.1.5 to be in range 192.168.1.1-192.168.1.10")
		}
	})

	t.Run("IP at start of range", func(t *testing.T) {
		if !IPInRange("192.168.1.1", "192.168.1.1-192.168.1.10") {
			t.Error("Expected 192.168.1.1 to be in range")
		}
	})

	t.Run("IP at end of range", func(t *testing.T) {
		if !IPInRange("192.168.1.10", "192.168.1.1-192.168.1.10") {
			t.Error("Expected 192.168.1.10 to be in range")
		}
	})

	t.Run("IP outside range", func(t *testing.T) {
		if IPInRange("192.168.1.11", "192.168.1.1-192.168.1.10") {
			t.Error("Expected 192.168.1.11 to NOT be in range")
		}
	})

	t.Run("IP below range", func(t *testing.T) {
		if IPInRange("192.168.0.255", "192.168.1.1-192.168.1.10") {
			t.Error("Expected 192.168.0.255 to NOT be in range")
		}
	})

	t.Run("Single IP range match", func(t *testing.T) {
		if !IPInRange("192.168.1.1", "192.168.1.1") {
			t.Error("Expected 192.168.1.1 to match single IP range")
		}
	})

	t.Run("Single IP range no match", func(t *testing.T) {
		if IPInRange("192.168.1.2", "192.168.1.1") {
			t.Error("Expected 192.168.1.2 to NOT match single IP range 192.168.1.1")
		}
	})

	t.Run("Invalid IP string", func(t *testing.T) {
		if IPInRange("not-an-ip", "192.168.1.1-192.168.1.10") {
			t.Error("Expected invalid IP to return false")
		}
	})

	t.Run("Invalid range string", func(t *testing.T) {
		if IPInRange("192.168.1.5", "invalid-range") {
			t.Error("Expected invalid range to return false")
		}
	})
}
