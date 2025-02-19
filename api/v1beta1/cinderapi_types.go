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
	"github.com/openstack-k8s-operators/lib-common/modules/common/service"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
)

// CinderAPITemplate defines the input parameters for the Cinder API service
type CinderAPITemplateCore struct {
	// Common input parameters for the Cinder API service
	CinderServiceTemplate `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// Replicas - Cinder API Replicas
	Replicas *int32 `json:"replicas"`

	// +kubebuilder:validation:Optional
	// Override, provides the ability to override the generated manifest of several child resources.
	Override APIOverrideSpec `json:"override,omitempty"`

	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// TLS - Parameters related to the TLS
	TLS tls.API `json:"tls,omitempty"`
}

// CinderAPITemplate defines the input parameters for the Cinder API service
type CinderAPITemplate struct {
	// +kubebuilder:validation:Required
	// ContainerImage - Cinder Container Image URL (will be set to environmental default if empty)
	ContainerImage string `json:"containerImage"`

	CinderAPITemplateCore `json:",inline"`
}

// APIOverrideSpec to override the generated manifest of several child resources.
type APIOverrideSpec struct {
	// Override configuration for the Service created to serve traffic to the cluster.
	// The key must be the endpoint type (public, internal)
	Service map[service.Endpoint]service.RoutedOverrideSpec `json:"service,omitempty"`
}

// CinderAPISpec defines the desired state of CinderAPI
type CinderAPISpec struct {
	// Common input parameters for all Cinder services
	CinderTemplate `json:",inline"`

	// Input parameters for the Cinder API service
	CinderAPITemplate `json:",inline"`

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
}

// CinderAPIStatus defines the observed state of CinderAPI
type CinderAPIStatus struct {
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// API endpoints
	APIEndpoints map[string]map[string]string `json:"apiEndpoints,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// ReadyCount of Cinder API instances
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	ReadyCount int32 `json:"readyCount"`

	// ServiceIDs
	ServiceIDs map[string]string `json:"serviceIDs,omitempty"`

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

// CinderAPI is the Schema for the cinderapis API
type CinderAPI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderAPISpec   `json:"spec,omitempty"`
	Status CinderAPIStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CinderAPIList contains a list of CinderAPI
type CinderAPIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CinderAPI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CinderAPI{}, &CinderAPIList{})
}

// IsReady - returns true if service is ready to serve requests
func (instance CinderAPI) IsReady() bool {
	return instance.Generation == instance.Status.ObservedGeneration &&
		instance.Status.ReadyCount == *instance.Spec.Replicas &&
		(instance.Status.Conditions.IsTrue(condition.DeploymentReadyCondition) ||
			(instance.Status.Conditions.IsFalse(condition.DeploymentReadyCondition) && *instance.Spec.Replicas == 0))
}

// GetLastAppliedTopology - Returns the LastAppliedTopology Set in the Status
func (instance CinderAPI) GetLastAppliedTopology() *topologyv1.TopoRef {
	return instance.Status.LastAppliedTopology
}
