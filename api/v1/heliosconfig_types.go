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
	// IPRange defines the IP address range for load balancer
	// +kubebuilder:validation:Required
	IPRange string `json:"ipRange"`

	// Service references the service to be load balanced
	// +kubebuilder:validation:Required
	Service string `json:"service"`

	// Protocol specifies the protocol (TCP/UDP)
	// +kubebuilder:validation:Enum=TCP;UDP
	// +kubebuilder:default:=TCP
	Protocol string `json:"protocol,omitempty"`

	// Port specifies the port to be load balanced
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:default:=80
	Port int32 `json:"port,omitempty"`

	// Method specifies the load balancing method
	// +kubebuilder:validation:Enum=RoundRobin;LeastConnection
	// +kubebuilder:default:=RoundRobin
	Method string `json:"method,omitempty"`

	// Interface specifies the network interface to use
	// +kubebuilder:validation:Required
	Interface string `json:"interface"`

	// ARPInterval specifies the interval in seconds between ARP announcements
	// +optional
	// +kubebuilder:default:=1
	// +kubebuilder:validation:Minimum=1
	ARPInterval int32 `json:"arpInterval,omitempty"`
}

// HeliosConfigStatus defines the observed state of HeliosConfig.
type HeliosConfigStatus struct {
	// AllocatedIPs is a map of service names to their allocated IPs
	AllocatedIPs map[string]string `json:"allocatedIPs,omitempty"`

	// NetworkInterface is the currently configured network interface
	NetworkInterface string `json:"networkInterface,omitempty"`

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
	ReasonInitializing       = "Initializing"
	ReasonNetworkConfigured  = "NetworkConfigured"
	ReasonNetworkError       = "NetworkError"
	ReasonIPAllocationError  = "IPAllocationError"
	ReasonARPAnnouncerActive = "ARPAnnouncerActive"
	ReasonARPAnnouncerFailed = "ARPAnnouncerFailed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
