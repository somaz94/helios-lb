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

	// 초기 health check
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
	// RLock 사용 (Lock 대신)
	lb.mu.RLock()
	backends := make(map[string][]*Backend)
	// 백엔드 목록 복사
	for service, bkends := range lb.backends {
		backends[service] = append([]*Backend{}, bkends...)
	}
	lb.mu.RUnlock()

	// 락 없이 health check 수행
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
