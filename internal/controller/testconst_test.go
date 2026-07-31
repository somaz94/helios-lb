package controller

import balancerv1 "github.com/somaz94/helios-lb/api/v1"

// Fixture values shared across the controller tests. Method and protocol
// values alias the api/v1 constants rather than repeating the literals, so a
// test can never drift from the enum the CRD actually accepts.
const (
	nsDefault = "default"
	nsAllowed = "allowed-ns"

	nameHelios1    = "helios-1"
	nameHelios2    = "helios-2"
	nameTestHelios = "test-helios"
	nameTestSvc    = "test-svc"
	nameSvcA       = "svc-a"
	nameSvc1       = "svc-1"

	methodRoundRobin = balancerv1.MethodRoundRobin
	methodWeighted   = balancerv1.MethodWeightedRoundRobin
	protocolTCP      = balancerv1.ProtocolTCP

	// Single-IP range: the allocator hands out exactly this address.
	testSingleIP = "172.16.0.100"

	ipRangeNarrow = "192.168.1.100-192.168.1.110"
	ipRangeWide   = "192.168.1.100-192.168.1.200"
	ipRange10Net  = "10.0.0.1-10.0.0.10"
)
