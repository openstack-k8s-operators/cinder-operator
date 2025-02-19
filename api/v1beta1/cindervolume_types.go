/*
Copyright 2022.

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

package v1beta1

import (
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
)

const (
	// Backend -
	Backend = "backend"
)

// CinderVolumeTemplate defines the input parameters for the Cinder Volume service
type CinderVolumeTemplateCore struct {
	// Common input parameters for the Cinder Volume service
	CinderServiceTemplate `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1
	// Replicas - Cinder Volume Replicas
	Replicas *int32 `json:"replicas"`
}

// CinderVolumeTemplate defines the input parameters for the Cinder Volume service
type CinderVolumeTemplate struct {
	// +kubebuilder:validation:Required
	// ContainerImage - Cinder Container Image URL (will be set to environmental default if empty)
	ContainerImage string `json:"containerImage"`

	CinderVolumeTemplateCore `json:",inline"`
}

// CinderVolumeSpec defines the desired state of CinderVolume
type CinderVolumeSpec struct {
	// Common input parameters for all Cinder services
	CinderTemplate `json:",inline"`

	// Input parameters for the Cinder Volume service
	CinderVolumeTemplate `json:",inline"`

	// +kubebuilder:validation:Required
	// DatabaseHostname - Cinder Database Hostname
	DatabaseHostname string `json:"databaseHostname"`

	// +kubebuilder:validation:Required
	// Secret containing RabbitMq transport URL
	TransportURLSecret string `json:"transportURLSecret"`

	// +kubebuilder:validation:Optional
	// ExtraMounts containing conf files and credentials
	ExtraMounts []CinderExtraVolMounts `json:"extraMounts,omitempty"`

	// +kubebuilder:validation:Required
	// ServiceAccount - service account name used internally to provide Cinder services the default SA name
	ServiceAccount string `json:"serviceAccount"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// TLS - Parameters related to the TLS
	TLS tls.Ca `json:"tls,omitempty"`
}

// CinderVolumeStatus defines the observed state of CinderVolume
type CinderVolumeStatus struct {
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// ReadyCount of Cinder Volume instances
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	ReadyCount int32 `json:"readyCount"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`

	// ObservedGeneration - the most recent generation observed for this service.
	// If the observed generation is different than the spec generation, then the
	// controller has not started processing the latest changes, and the status
	// and its conditions are likely stale.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// LastAppliedTopology - the last applied Topology
	LastAppliedTopology *topologyv1.TopoRef `json:"lastAppliedTopology,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="NetworkAttachments",type="string",JSONPath=".status.networkAttachments",description="NetworkAttachments"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// CinderVolume is the Schema for the cindervolumes API
type CinderVolume struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderVolumeSpec   `json:"spec,omitempty"`
	Status CinderVolumeStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CinderVolumeList contains a list of CinderVolume
type CinderVolumeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CinderVolume `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CinderVolume{}, &CinderVolumeList{})
}

// IsReady - returns true if service is ready to serve requests
func (instance CinderVolume) IsReady() bool {
	return instance.Generation == instance.Status.ObservedGeneration &&
		instance.Status.ReadyCount == *instance.Spec.Replicas &&
		(instance.Status.Conditions.IsTrue(condition.DeploymentReadyCondition) ||
			(instance.Status.Conditions.IsFalse(condition.DeploymentReadyCondition) && *instance.Spec.Replicas == 0))
}

// BackendName - returns the backend name of a CinderVolume instance based on the labels
func (instance CinderVolume) BackendName() string {
	return instance.Labels[Backend]
}

// GetLastAppliedTopology - Returns the LastAppliedTopology Set in the Status
func (instance CinderVolume) GetLastAppliedTopology() *topologyv1.TopoRef {
	return instance.Status.LastAppliedTopology
}
