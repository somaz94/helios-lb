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

func TestIPAllocator_CIDR(t *testing.T) {
	t.Run("CIDR /30 allocates usable IPs", func(t *testing.T) {
		allocator := NewIPAllocator()
		ip1, err := allocator.AllocateIP("192.168.1.0/30")
		if err != nil {
			t.Fatalf("Failed to allocate IP from CIDR: %v", err)
		}
		// /30 has 4 addresses, skip network (0) and broadcast (3), usable: .1 and .2
		if ip1 != "192.168.1.1" {
			t.Errorf("Expected 192.168.1.1, got %s", ip1)
		}

		ip2, err := allocator.AllocateIP("192.168.1.0/30")
		if err != nil {
			t.Fatalf("Failed to allocate second IP from CIDR: %v", err)
		}
		if ip2 != "192.168.1.2" {
			t.Errorf("Expected 192.168.1.2, got %s", ip2)
		}

		// Should be exhausted
		_, err = allocator.AllocateIP("192.168.1.0/30")
		if err == nil {
			t.Error("Expected error when CIDR range is exhausted")
		}
	})

	t.Run("CIDR /28 allocates multiple IPs", func(t *testing.T) {
		allocator := NewIPAllocator()
		// /28 = 16 addresses, usable: .1 to .14
		var ips []string
		for i := 0; i < 14; i++ {
			ip, err := allocator.AllocateIP("10.0.0.0/28")
			if err != nil {
				t.Fatalf("Failed to allocate IP %d: %v", i, err)
			}
			ips = append(ips, ip)
		}
		if ips[0] != "10.0.0.1" {
			t.Errorf("Expected first usable IP 10.0.0.1, got %s", ips[0])
		}
		if ips[13] != "10.0.0.14" {
			t.Errorf("Expected last usable IP 10.0.0.14, got %s", ips[13])
		}
	})

	t.Run("Invalid CIDR format", func(t *testing.T) {
		allocator := NewIPAllocator()
		_, err := allocator.AllocateIP("192.168.1.0/abc")
		if err == nil {
			t.Error("Expected error for invalid CIDR")
		}
	})
}

func TestIPInRange_CIDR(t *testing.T) {
	t.Run("IP within CIDR", func(t *testing.T) {
		if !IPInRange("192.168.1.50", "192.168.1.0/24") {
			t.Error("Expected 192.168.1.50 to be in 192.168.1.0/24")
		}
	})

	t.Run("IP outside CIDR", func(t *testing.T) {
		if IPInRange("192.168.2.1", "192.168.1.0/24") {
			t.Error("Expected 192.168.2.1 to NOT be in 192.168.1.0/24")
		}
	})

	t.Run("Network address in CIDR", func(t *testing.T) {
		if !IPInRange("192.168.1.0", "192.168.1.0/24") {
			t.Error("Expected 192.168.1.0 to be in 192.168.1.0/24")
		}
	})

	t.Run("Broadcast address in CIDR", func(t *testing.T) {
		if !IPInRange("192.168.1.255", "192.168.1.0/24") {
			t.Error("Expected 192.168.1.255 to be in 192.168.1.0/24")
		}
	})

	t.Run("Invalid CIDR in range check", func(t *testing.T) {
		if IPInRange("192.168.1.1", "invalid/cidr") {
			t.Error("Expected false for invalid CIDR")
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
