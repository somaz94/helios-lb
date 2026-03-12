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

// parseIPRange parses IP range string (e.g., "192.168.1.100-192.168.1.110" or "192.168.1.100")
// Returns start and end IPs without modifying shared state.
func parseIPRange(ipRange string) (start, end net.IP, err error) {
	// First try single IP
	if ip := net.ParseIP(strings.TrimSpace(ipRange)); ip != nil {
		return ip, ip, nil
	}

	// Try parsing IP range
	parts := strings.Split(ipRange, "-")
	if len(parts) != 2 {
		return nil, nil, fmt.Errorf("invalid IP range format: %s", ipRange)
	}

	start = net.ParseIP(strings.TrimSpace(parts[0]))
	end = net.ParseIP(strings.TrimSpace(parts[1]))

	if start == nil || end == nil {
		return nil, nil, fmt.Errorf("invalid IP addresses in range: %s", ipRange)
	}

	return start, end, nil
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

// ReleaseIP releases an allocated IP
func (a *IPAllocator) ReleaseIP(ip string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.used, ip)
}
