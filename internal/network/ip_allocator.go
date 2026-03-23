package network

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
)

// IPAllocator handles IP address allocation
type IPAllocator struct {
	mu   sync.Mutex
	used map[string]bool
}

// NewIPAllocator creates a new IPAllocator
func NewIPAllocator() *IPAllocator {
	return &IPAllocator{
		used: make(map[string]bool),
	}
}

// normalizeIP returns a consistent representation of an IP address.
// IPv4 addresses are returned as 4-byte slices, IPv6 as 16-byte slices.
// This ensures bytes.Compare works correctly across all IP comparisons.
func normalizeIP(ip net.IP) net.IP {
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip.To16()
}

// parseIPRange parses IP range string in the following formats:
//   - Single IP: "192.168.1.100" or "fd00::1"
//   - Range: "192.168.1.100-192.168.1.110" or "fd00::1-fd00::ff"
//   - CIDR: "192.168.1.0/24" or "fd00::/120"
//
// Supports both IPv4 and IPv6 addresses.
// Returns start and end IPs without modifying shared state.
func parseIPRange(ipRange string) (start, end net.IP, err error) {
	trimmed := strings.TrimSpace(ipRange)

	// Try CIDR notation (e.g., "192.168.1.0/24" or "fd00::/120")
	if strings.Contains(trimmed, "/") {
		_, ipNet, err := net.ParseCIDR(trimmed)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid CIDR format: %s", ipRange)
		}
		start := normalizeIP(ipNet.IP)
		// Calculate the last IP in the CIDR range
		end := make(net.IP, len(start))
		for i := range start {
			end[i] = start[i] | ^ipNet.Mask[i]
		}
		// Skip network address (first) and broadcast address (last) for IPv4 subnets > /31
		ones, bits := ipNet.Mask.Size()
		if bits == 32 && bits-ones > 1 {
			start = incrementIP(start)
			end = decrementIP(end)
		}
		return start, end, nil
	}

	// Try single IP
	if ip := net.ParseIP(trimmed); ip != nil {
		normalized := normalizeIP(ip)
		return normalized, normalized, nil
	}

	// Try parsing IP range (e.g., "192.168.1.100-192.168.1.110" or "fd00::1-fd00::ff")
	parts := strings.Split(trimmed, "-")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid IP range format: %s", ipRange)
	}

	start = net.ParseIP(strings.TrimSpace(parts[0]))
	end = net.ParseIP(strings.TrimSpace(parts[1]))

	if start == nil || end == nil {
		return nil, nil, fmt.Errorf("invalid IP addresses in range: %s", ipRange)
	}

	return normalizeIP(start), normalizeIP(end), nil
}

// incrementIP returns the next IP address (network helper for CIDR)
func incrementIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}

// decrementIP returns the previous IP address (network helper for CIDR)
func decrementIP(ip net.IP) net.IP {
	prev := make(net.IP, len(ip))
	copy(prev, ip)
	for i := len(prev) - 1; i >= 0; i-- {
		prev[i]--
		if prev[i] != 0xFF {
			break
		}
	}
	return prev
}

// nextIP returns the next IP address
func (a *IPAllocator) nextIP(ip net.IP) net.IP {
	next := make(net.IP, len(ip))
	copy(next, ip)
	for i := len(next) - 1; i >= 0; i-- {
		next[i]++
		if next[i] != 0 {
			break
		}
	}
	return next
}

// AllocateIP allocates an available IP from the range
func (a *IPAllocator) AllocateIP(ipRange string) (string, error) {
	start, end, err := parseIPRange(ipRange)
	if err != nil {
		return "", err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// If the requested IP is a single IP within the range
	if start.Equal(end) {
		ipStr := start.String()
		// Even if it's already in use, return the same IP
		a.used[ipStr] = true
		return ipStr, nil
	}

	// Allocate IP from the range using bytes comparison instead of string comparison
	for ip := start; bytes.Compare(ip, end) <= 0; ip = a.nextIP(ip) {
		ipStr := ip.String()
		if !a.used[ipStr] {
			a.used[ipStr] = true
			return ipStr, nil
		}
	}

	return "", fmt.Errorf("no available IPs in range %s", ipRange)
}

// IPInRange checks if the given IP string falls within the specified range.
// Supports single IP, range format, and CIDR notation.
func IPInRange(ip string, ipRange string) bool {
	target := net.ParseIP(strings.TrimSpace(ip))
	if target == nil {
		return false
	}

	// Fast path for CIDR: use net.Contains directly
	if strings.Contains(ipRange, "/") {
		_, ipNet, err := net.ParseCIDR(strings.TrimSpace(ipRange))
		if err != nil {
			return false
		}
		return ipNet.Contains(target)
	}

	start, end, err := parseIPRange(ipRange)
	if err != nil {
		return false
	}
	normalized := normalizeIP(target)
	return bytes.Compare(normalized, start) >= 0 && bytes.Compare(normalized, end) <= 0
}

// MarkUsed marks an IP as used without allocating it.
// This is used to prevent conflicts with IPs allocated by other HeliosConfigs.
func (a *IPAllocator) MarkUsed(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.used[ip] = true
}

// ReleaseIP releases an allocated IP
func (a *IPAllocator) ReleaseIP(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.used, ip)
}
