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

// CinderAPISpec defines the desired state of CinderAPI
type CinderAPISpec struct {
	// CR name of managing controller object to identify the config maps
	ManagingCrName string `json:"managingCrName,omitempty"`
	// Cinder Database Hostname String
	DatabaseHostname string `json:"databaseHostname,omitempty"`
	// Cinder Scheduler Container Image URL
	ContainerImage string `json:"containerImage,omitempty"`
	// Cinder API Replicas
	Replicas int32 `json:"replicas"`
	// Secret containing: CinderPassword, TransportURL
	CinderSecret string `json:"cinderSecret,omitempty"`
	// Secret containing: NovaPassword
	NovaSecret string `json:"novaSecret,omitempty"`
}

// CinderAPIStatus defines the observed state of CinderAPI
type CinderAPIStatus struct {
	// hashes of Secrets, CMs
	Hashes []Hash `json:"hashes,omitempty"`
	// CinderAPIHash deployment hash
	CinderAPIHash string `json:"cinderAPIHash"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CinderAPI is the Schema for the cinderapis API
type CinderAPI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderAPISpec   `json:"spec,omitempty"`
	Status CinderAPIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CinderAPIList contains a list of CinderAPI
type CinderAPIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CinderAPI `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CinderAPI{}, &CinderAPIList{})
}
