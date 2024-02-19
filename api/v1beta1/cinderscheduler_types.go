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
)

// CinderSchedulerTemplate defines the input parameters for the Cinder Scheduler service
type CinderSchedulerTemplate struct {
	// Common input parameters for the Cinder Scheduler service
	CinderServiceTemplate `json:",inline"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// Replicas - Cinder Scheduler Replicas
	Replicas *int32 `json:"replicas"`
}

// CinderSchedulerSpec defines the desired state of CinderScheduler
type CinderSchedulerSpec struct {
	// Common input parameters for all Cinder services
	CinderTemplate `json:",inline"`

	// Input parameters for the Cinder Scheduler service
	CinderSchedulerTemplate `json:",inline"`

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

// CinderSchedulerStatus defines the observed state of CinderScheduler
type CinderSchedulerStatus struct {
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// ReadyCount of Cinder Scheduler instances
	ReadyCount int32 `json:"readyCount,omitempty"`

	// NetworkAttachments status of the deployment pods
	NetworkAttachments map[string][]string `json:"networkAttachments,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="NetworkAttachments",type="string",JSONPath=".status.networkAttachments",description="NetworkAttachments"
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// CinderScheduler is the Schema for the cinderschedulers API
type CinderScheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderSchedulerSpec   `json:"spec,omitempty"`
	Status CinderSchedulerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CinderSchedulerList contains a list of CinderScheduler
type CinderSchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CinderScheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CinderScheduler{}, &CinderSchedulerList{})
}

// IsReady - returns true if service is ready to serve requests
func (instance CinderScheduler) IsReady() bool {
	return instance.Status.ReadyCount == *instance.Spec.Replicas
}
