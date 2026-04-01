package network

import (
	"bytes"
	"fmt"
	"net"
	"strings"
)

// NormalizeIP returns a consistent representation of an IP address.
// IPv4 addresses are returned as 4-byte slices, IPv6 as 16-byte slices.
// This ensures bytes.Compare works correctly across all IP comparisons.
func NormalizeIP(ip net.IP) net.IP {
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip.To16()
}

// ParseIPRange parses IP range string in the following formats:
//   - Single IP: "192.168.1.100" or "fd00::1"
//   - Range: "192.168.1.100-192.168.1.110" or "fd00::1-fd00::ff"
//   - CIDR: "192.168.1.0/24" or "fd00::/120"
//
// Supports both IPv4 and IPv6 addresses.
// Returns normalized start and end IPs.
func ParseIPRange(ipRange string) (start, end net.IP, err error) {
	trimmed := strings.TrimSpace(ipRange)

	// Try CIDR notation (e.g., "192.168.1.0/24" or "fd00::/120")
	if strings.Contains(trimmed, "/") {
		_, ipNet, err := net.ParseCIDR(trimmed)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid CIDR format: %s", ipRange)
		}
		start := NormalizeIP(ipNet.IP)
		// Calculate the last IP in the CIDR range
		end := make(net.IP, len(start))
		for i := range start {
			end[i] = start[i] | ^ipNet.Mask[i]
		}
		// Skip network address (first) and broadcast address (last) for IPv4 subnets > /31
		ones, bits := ipNet.Mask.Size()
		if bits == 32 && bits-ones > 1 {
			start = IncrementIP(start)
			end = DecrementIP(end)
		}
		return start, end, nil
	}

	// Try single IP
	if ip := net.ParseIP(trimmed); ip != nil {
		normalized := NormalizeIP(ip)
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

	return NormalizeIP(start), NormalizeIP(end), nil
}

// CompareIPs compares two IPs byte-by-byte. Returns -1, 0, or 1.
func CompareIPs(a, b net.IP) int {
	a = NormalizeIP(a)
	b = NormalizeIP(b)
	return bytes.Compare(a, b)
}

// IncrementIP returns the next IP address.
func IncrementIP(ip net.IP) net.IP {
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

// DecrementIP returns the previous IP address.
func DecrementIP(ip net.IP) net.IP {
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
