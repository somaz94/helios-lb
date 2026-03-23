package loadbalancer

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// HealthCheckOptions holds configurable health check parameters.
type HealthCheckOptions struct {
	Timeout  time.Duration
	Protocol string // "TCP" or "HTTP"
	HTTPPath string // path for HTTP checks (e.g., "/healthz")
}

// DefaultHealthCheckOptions returns the default health check options.
func DefaultHealthCheckOptions() HealthCheckOptions {
	return HealthCheckOptions{
		Timeout:  time.Second,
		Protocol: "TCP",
	}
}

func (lb *LoadBalancer) healthCheckLoop() {
	defer lb.wg.Done()

	ticker := time.NewTicker(lb.config.CheckInterval)
	defer ticker.Stop()

	// Initial health check
	lb.checkWg.Add(1)
	go func() {
		lb.doHealthCheck()
		lb.checkWg.Done()
	}()

	for {
		select {
		case <-lb.stopCh:
			lb.checkWg.Wait()
			return
		case <-ticker.C:
			lb.checkWg.Add(1)
			go func() {
				lb.doHealthCheck()
				lb.checkWg.Done()
			}()
		}
	}
}

func (lb *LoadBalancer) doHealthCheck() {
	// Use RLock instead of Lock
	lb.mu.RLock()
	backends := make(map[string][]*Backend)
	for service, bkends := range lb.backends {
		backends[service] = append([]*Backend{}, bkends...)
	}
	opts := lb.config.HealthCheckOpts
	lb.mu.RUnlock()

	for _, bkends := range backends {
		for _, backend := range bkends {
			healthy := checkBackendHealth(backend, opts)
			backend.SetHealthy(healthy)
		}
	}
}

// checkBackendHealth checks a single backend using the configured protocol.
func checkBackendHealth(backend *Backend, opts HealthCheckOptions) bool {
	address := net.JoinHostPort(backend.Address, fmt.Sprintf("%d", backend.Port))

	switch opts.Protocol {
	case "HTTP":
		return checkHTTP(address, opts)
	default: // TCP
		return checkTCP(address, opts.Timeout)
	}
}

func checkTCP(address string, timeout time.Duration) bool {
	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", address)
	if err != nil {
		return false
	}
	if conn != nil {
		conn.Close()
		return true
	}
	return false
}

func checkHTTP(address string, opts HealthCheckOptions) bool {
	path := opts.HTTPPath
	if path == "" {
		path = "/"
	}
	url := fmt.Sprintf("http://%s%s", address, path)
	client := &http.Client{Timeout: opts.Timeout}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}
