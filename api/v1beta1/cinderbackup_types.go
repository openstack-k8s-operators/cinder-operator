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

// CinderBackupTemplate defines the input parameters for the Cinder Backup service
type CinderBackupTemplateCore struct {
	// Common input parameters for the Cinder Backup service
	CinderServiceTemplate `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// Replicas - Cinder Backup Replicas
	Replicas *int32 `json:"replicas"`
}

// CinderBackupTemplate defines the input parameters for the Cinder Backup service
type CinderBackupTemplate struct {
	// +kubebuilder:validation:Required
	// ContainerImage - Cinder Container Image URL (will be set to environmental default if empty)
	ContainerImage string `json:"containerImage"`

	CinderBackupTemplateCore `json:",inline"`
}

// CinderBackupSpec defines the desired state of CinderBackup
type CinderBackupSpec struct {
	// Common input parameters for all Cinder services
	CinderTemplate `json:",inline"`

	// Input parameters for the Cinder Backup service
	CinderBackupTemplate `json:",inline"`

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

// CinderBackupStatus defines the observed state of CinderBackup
type CinderBackupStatus struct {
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// ReadyCount of Cinder Backup instances
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

// CinderBackup is the Schema for the cinderbackups API
type CinderBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderBackupSpec   `json:"spec,omitempty"`
	Status CinderBackupStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CinderBackupList contains a list of CinderBackup
type CinderBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CinderBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CinderBackup{}, &CinderBackupList{})
}

// IsReady - returns true if service is ready to serve requests
func (instance CinderBackup) IsReady() bool {
	return instance.Generation == instance.Status.ObservedGeneration &&
		instance.Status.ReadyCount == *instance.Spec.Replicas &&
		(instance.Status.Conditions.IsTrue(condition.DeploymentReadyCondition) ||
			(instance.Status.Conditions.IsFalse(condition.DeploymentReadyCondition) && *instance.Spec.Replicas == 0))
}

// GetLastAppliedTopology - Returns the LastAppliedTopology Set in the Status
func (instance CinderBackup) GetLastAppliedTopology() *topologyv1.TopoRef {
	return instance.Status.LastAppliedTopology
}
