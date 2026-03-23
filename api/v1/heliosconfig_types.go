/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HeliosConfigSpec defines the desired state of HeliosConfig.
type HeliosConfigSpec struct {
	// IPRange defines the IP address range for load balancer.
	// Supports single IP ("192.168.1.100"), range ("192.168.1.100-192.168.1.200"),
	// and CIDR notation ("192.168.1.0/24").
	// +kubebuilder:validation:Required
	IPRange string `json:"ipRange"`

	// Service references the service to be load balanced
	// +optional
	Service string `json:"service,omitempty"`

	// Protocol specifies the protocol (TCP/UDP)
	// +kubebuilder:validation:Enum=TCP;UDP
	// +kubebuilder:default:=TCP
	Protocol string `json:"protocol,omitempty"`

	// Ports specifies the ports to be load balanced
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	// +kubebuilder:default:={{port: 80}}
	Ports []PortConfig `json:"ports,omitempty"`

	// Method specifies the load balancing method
	// +kubebuilder:validation:Enum=RoundRobin;LeastConnection;WeightedRoundRobin;IPHash;Random
	// +kubebuilder:default:=RoundRobin
	Method string `json:"method,omitempty"`

	// HealthCheck configures backend health checking
	// +optional
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`
}

// HealthCheckConfig defines the health check parameters for backends
type HealthCheckConfig struct {
	// Enabled enables or disables health checking
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled"`

	// IntervalSeconds is the interval between health checks in seconds
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=300
	// +kubebuilder:default:=5
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`

	// TimeoutMs is the health check timeout in milliseconds
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=30000
	// +kubebuilder:default:=1000
	TimeoutMs int32 `json:"timeoutMs,omitempty"`

	// Protocol specifies the health check protocol
	// +kubebuilder:validation:Enum=TCP;HTTP
	// +kubebuilder:default:=TCP
	Protocol string `json:"protocol,omitempty"`

	// HTTPPath is the HTTP path for HTTP health checks (only used when protocol is HTTP)
	// +optional
	HTTPPath string `json:"httpPath,omitempty"`
}

// PortConfig defines the configuration for a port
type PortConfig struct {
	// Port number
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`

	// Protocol for this specific port (optional, defaults to spec.Protocol)
	// +kubebuilder:validation:Enum=TCP;UDP
	// +optional
	Protocol string `json:"protocol,omitempty"`
}

// HeliosConfigStatus defines the observed state of HeliosConfig.
type HeliosConfigStatus struct {
	// AllocatedIPs is a map of service names to their allocated IPs
	AllocatedIPs map[string]string `json:"allocatedIPs,omitempty"`

	// State represents the current state of the load balancer
	// +kubebuilder:validation:Enum=Pending;Active;Failed
	State string `json:"state,omitempty"`

	// Message provides additional information about the state
	Message string `json:"message,omitempty"`

	// LastUpdated is the timestamp of the last status update
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`

	// Conditions represent the latest available observations of the HeliosConfig's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Phase represents the current state of the HeliosConfig
	// +optional
	Phase string `json:"phase,omitempty"`
}

// HeliosConfig Constants
const (
	// State constants
	StatePending = "Pending"
	StateActive  = "Active"
	StateFailed  = "Failed"

	// Condition types
	ConditionTypeReady     = "Ready"
	ConditionTypeAvailable = "Available"
	ConditionTypeDegraded  = "Degraded"

	// Condition reasons
	ReasonInitializing      = "Initializing"
	ReasonNetworkConfigured = "NetworkConfigured"
	ReasonNetworkError      = "NetworkError"
	ReasonIPAllocationError = "IPAllocationError"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// HeliosConfig is the Schema for the heliosconfigs API.
type HeliosConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HeliosConfigSpec   `json:"spec,omitempty"`
	Status HeliosConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HeliosConfigList contains a list of HeliosConfig.
type HeliosConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HeliosConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HeliosConfig{}, &HeliosConfigList{})
}
