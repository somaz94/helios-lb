package network

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

// IPAllocator handles IP address allocation
type IPAllocator struct {
	mu    sync.Mutex
	used  map[string]bool
	start net.IP
	end   net.IP
}

// NewIPAllocator creates a new IPAllocator
func NewIPAllocator() *IPAllocator {
	return &IPAllocator{
		used: make(map[string]bool),
	}
}

// parseIPRange parses IP range string (e.g., "192.168.1.100-192.168.1.110" or "192.168.1.100")
func (a *IPAllocator) parseIPRange(ipRange string) error {
	// 먼저 단일 IP 시도
	if ip := net.ParseIP(strings.TrimSpace(ipRange)); ip != nil {
		a.start = ip
		a.end = ip
		return nil
	}

	// IP 범위 파싱 시도
	parts := strings.Split(ipRange, "-")
	if len(parts) != 2 {
		return fmt.Errorf("invalid IP range format: %s", ipRange)
	}

	a.start = net.ParseIP(strings.TrimSpace(parts[0]))
	a.end = net.ParseIP(strings.TrimSpace(parts[1]))

	if a.start == nil || a.end == nil {
		return fmt.Errorf("invalid IP addresses in range: %s", ipRange)
	}

	return nil
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
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.parseIPRange(ipRange); err != nil {
		return "", err
	}

	// 요청된 IP가 범위 내의 단일 IP인 경우
	if a.start.Equal(a.end) {
		ipStr := a.start.String()
		// 이미 사용 중이더라도 같은 IP를 반환
		a.used[ipStr] = true
		return ipStr, nil
	}

	// 범위에서 IP 할당
	for ip := a.start; ip.String() <= a.end.String(); ip = a.nextIP(ip) {
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
