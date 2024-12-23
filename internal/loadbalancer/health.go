package loadbalancer

import (
	"fmt"
	"net"
	"time"
)

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
	// Copy backend list
	for service, bkends := range lb.backends {
		backends[service] = append([]*Backend{}, bkends...)
	}
	lb.mu.RUnlock()

	// Perform health check without locking
	for _, bkends := range backends {
		for _, backend := range bkends {
			d := net.Dialer{
				Timeout: time.Millisecond * 10,
			}
			conn, err := d.Dial("tcp", fmt.Sprintf("%s:%d", backend.Address, backend.Port))
			if err != nil {
				backend.SetHealthy(false)
				continue
			}
			if conn != nil {
				conn.Close()
				backend.SetHealthy(true)
			}
		}
	}
}
