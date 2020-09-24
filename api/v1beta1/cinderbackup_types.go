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

// CinderBackupSpec defines the desired state of CinderBackup
type CinderBackupSpec struct {
	// CR name of managing controller object to identify the config maps
	ManagingCrName string `json:"managingCrName,omitempty"`
	// Cinder Backup node selector
	NodeSelectorRoleName string `json:"nodeSelectorRoleName,omitempty"`
	// Cinder Database Hostname String
	DatabaseHostname string `json:"databaseHostname,omitempty"`
	// Cinder Backup Container Image URL
	ContainerImage string `json:"containerImage,omitempty"`
	// Cinder Backup Replicas
	Replicas int32 `json:"replicas"`
	// Secret containing: CinderPassword, TransportURL
	CinderSecret string `json:"cinderSecret,omitempty"`
	// Secret containing: NovaPassword
	NovaSecret string `json:"novaSecret,omitempty"`
}

// CinderBackupStatus defines the observed state of CinderBackup
type CinderBackupStatus struct {
	// hashes of Secrets, CMs
	Hashes []Hash `json:"hashes,omitempty"`
	// CinderBackupHash deployment hash
	CinderBackupHash string `json:"cinderBackupHash"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// CinderBackup is the Schema for the cinderbackups API
type CinderBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderBackupSpec   `json:"spec,omitempty"`
	Status CinderBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CinderBackupList contains a list of CinderBackup
type CinderBackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CinderBackup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CinderBackup{}, &CinderBackupList{})
}
