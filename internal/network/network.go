package network

type NetworkManager struct {
	ipAllocator *IPAllocator
}

func NewNetworkManager() *NetworkManager {
	return &NetworkManager{
		ipAllocator: NewIPAllocator(),
	}
}

// AllocateIP allocates an IP from the given range
func (nm *NetworkManager) AllocateIP(ipRange string) (string, error) {
	return nm.ipAllocator.AllocateIP(ipRange)
}

// MarkUsed marks an IP as used to prevent conflict during allocation.
func (nm *NetworkManager) MarkUsed(ip string) {
	nm.ipAllocator.MarkUsed(ip)
}

// ReleaseIP releases an IP
func (nm *NetworkManager) ReleaseIP(ip string) {
	nm.ipAllocator.ReleaseIP(ip)
}
