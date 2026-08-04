package network

import (
	"bytes"
	"errors"
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

// ErrIPUnavailable reports that a specifically requested IP cannot be handed
// out. Callers surface it rather than substituting a different address, so a
// user who asked for one IP is never silently given another.
var ErrIPUnavailable = errors.New("requested IP is unavailable")

// AllocateSpecificIP allocates exactly the requested IP from the range.
//
// It is the path taken when a Service asks for an address through
// spec.loadBalancerIP. The request is honored or it fails: the caller must not
// fall back to an arbitrary address, because a silent substitution is the
// hardest kind of misconfiguration to notice.
//
// The IP must be allocatable, not merely contained in the range — for an IPv4
// CIDR that excludes the network and broadcast addresses, which AllocateIP
// never hands out either.
func (a *IPAllocator) AllocateSpecificIP(ipRange, requested string) (string, error) {
	target := net.ParseIP(strings.TrimSpace(requested))
	if target == nil {
		return "", fmt.Errorf("%w: %q is not a valid IP", ErrIPUnavailable, requested)
	}

	if !IPAllocatable(requested, ipRange) {
		return "", fmt.Errorf("%w: %s is not an allocatable address in range %s", ErrIPUnavailable, requested, ipRange)
	}

	ipStr := NormalizeIP(target).String()

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.used[ipStr] {
		return "", fmt.Errorf("%w: %s is already allocated", ErrIPUnavailable, ipStr)
	}

	a.used[ipStr] = true
	return ipStr, nil
}

// IPAllocatable reports whether ip is an address the allocator would hand out
// for ipRange.
//
// This is stricter than IPInRange, which answers plain containment: for an IPv4
// CIDR wider than /31, IPInRange accepts the network and broadcast addresses
// while this rejects them, matching what AllocateIP actually yields.
func IPAllocatable(ip string, ipRange string) bool {
	target := net.ParseIP(strings.TrimSpace(ip))
	if target == nil {
		return false
	}

	start, end, err := ParseIPRange(ipRange)
	if err != nil {
		return false
	}

	normalized := NormalizeIP(target)
	return bytes.Compare(normalized, start) >= 0 && bytes.Compare(normalized, end) <= 0
}

// IPInRange checks if the given IP string falls within the specified range.
// Supports single IP, range format, and CIDR notation.
//
// This answers containment only. Use IPAllocatable when the question is whether
// an address can actually be handed out.
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
