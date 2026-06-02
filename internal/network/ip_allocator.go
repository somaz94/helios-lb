package network

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
)

// defaultMaxScan bounds AllocateIP's linear scan. 65536 covers a full IPv4 /16
// (the largest realistic LB IP pool); without it, a very large range such as an
// IPv6 /64 would scan an effectively unbounded address space while holding a.mu.
const defaultMaxScan = 1 << 16

// IPAllocator handles IP address allocation
type IPAllocator struct {
	mu      sync.Mutex
	used    map[string]bool
	maxScan int
}

// NewIPAllocator creates a new IPAllocator
func NewIPAllocator() *IPAllocator {
	return &IPAllocator{
		used:    make(map[string]bool),
		maxScan: defaultMaxScan,
	}
}

// AllocateIP allocates an available IP from the range
func (a *IPAllocator) AllocateIP(ipRange string) (string, error) {
	start, end, err := ParseIPRange(ipRange)
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

	// Allocate IP from the range using bytes comparison instead of string comparison.
	// The scan is bounded by a.maxScan so a very large range (e.g. an IPv6 /64) cannot
	// hold a.mu while scanning an effectively unbounded address space.
	scanned := 0
	for ip := start; bytes.Compare(ip, end) <= 0; ip = IncrementIP(ip) {
		if scanned >= a.maxScan {
			return "", fmt.Errorf("no available IP found in range %s within scan limit (%d addresses)", ipRange, a.maxScan)
		}
		scanned++
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

	start, end, err := ParseIPRange(ipRange)
	if err != nil {
		return false
	}
	normalized := NormalizeIP(target)
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
