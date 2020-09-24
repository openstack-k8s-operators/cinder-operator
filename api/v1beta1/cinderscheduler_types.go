/*
Copyright 2020 Red Hat

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CinderSchedulerSpec defines the desired state of CinderScheduler
type CinderSchedulerSpec struct {
	// CR name of managing controller object to identify the config maps
	ManagingCrName string `json:"managingCrName,omitempty"`
	// Cinder Database Hostname String
	DatabaseHostname string `json:"databaseHostname,omitempty"`
	// Cinder Scheduler Container Image URL
	ContainerImage string `json:"containerImage,omitempty"`
	// Cinder Scheduler Replicas
	Replicas int32 `json:"replicas"`
	// Secret containing: CinderPassword, TransportURL
	CinderSecret string `json:"cinderSecret,omitempty"`
	// Secret containing: NovaPassword
	NovaSecret string `json:"novaSecret,omitempty"`
}

// CinderSchedulerStatus defines the observed state of CinderScheduler
type CinderSchedulerStatus struct {
	// hashes of Secrets, CMs
	Hashes []Hash `json:"hashes,omitempty"`
	// CinderSchedulerHash deployment hash
	CinderSchedulerHash string `json:"cinderSchedulerHash"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CinderScheduler is the Schema for the cinderschedulers API
type CinderScheduler struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderSchedulerSpec   `json:"spec,omitempty"`
	Status CinderSchedulerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CinderSchedulerList contains a list of CinderScheduler
type CinderSchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CinderScheduler `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CinderScheduler{}, &CinderSchedulerList{})
}
