package controller

// Structured logging key constants for consistent log output.
const (
	LogKeyService       = "service"
	LogKeyNamespace     = "namespace"
	LogKeyIP            = "ip"
	LogKeyIPRange       = "ipRange"
	LogKeyConfig        = "config"
	LogKeyPhase         = "phase"
	LogKeyMaxAlloc      = "maxAllocations"
	LogKeyCurrentAlloc  = "currentAllocations"
	LogKeyError         = "error"
	LogKeyAllocatedIPs  = "allocatedIPs"
	LogKeyServiceCount  = "serviceCount"
	LogKeyReconcileTime = "reconcileTimeMs"
	LogKeyConflictIP    = "conflictIP"
	LogKeyConflictOwner = "conflictOwner"
)
