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

// CinderSpec defines the desired state of Cinder
type CinderSpec struct {
	// Cinder Database Hostname String
	DatabaseHostname string `json:"databaseHostname,omitempty"`
	// Cinder API Container Image URL
	CinderAPIContainerImage string `json:"cinderAPIContainerImage,omitempty"`
	// Cinder Backup Container Image URL
	CinderBackupContainerImage string `json:"cinderBackupContainerImage,omitempty"`
	// Cinder Scheduler Container Image URL
	CinderSchedulerContainerImage string `json:"cinderSchedulerContainerImage,omitempty"`
	// Cinder API Replicas
	CinderAPIReplicas int32 `json:"cinderAPIReplicas"`
	// Cinder Backup Replicas
	CinderBackupReplicas int32 `json:"cinderBackupReplicas"`
	// Cinder Backup node selector
	CinderBackupNodeSelectorRoleName string `json:"cinderBackupNodeSelectorRoleName,omitempty"`
	// Cinder Scheduler Replicas
	CinderSchedulerReplicas int32 `json:"cinderSchedulerReplicas"`
	// Secret containing: CinderPassword, TransportURL
	CinderSecret string `json:"cinderSecret,omitempty"`
	// Secret containing: NovaPassword
	NovaSecret string `json:"novaSecret,omitempty"`
	// Cells to create
	CinderVolumes []Volume `json:"cinderVolumes,omitempty"`
}

// Volume defines cinder volume configuration parameters
type Volume struct {
	// Name of cinder volume service
	Name string `json:"name,omitempty"`
	// Cinder Volume Container Image URL
	CinderVolumeContainerImage string `json:"cinderVolumeContainerImage,omitempty"`
	// Cinder Volume Replicas
	CinderVolumeReplicas int32 `json:"cinderVolumeReplicas"`
	// Cinder Volume node selector
	CinderVolumeNodeSelectorRoleName string `json:"cinderVolumeNodeSelectorRoleName,omitempty"`
}

// CinderStatus defines the observed state of Cinder
type CinderStatus struct {
	// DbSyncHash db sync hash
	DbSyncHash string `json:"dbSyncHash"`
	// API endpoint
	APIEndpoint string `json:"apiEndpoint"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Cinder is the Schema for the cinders API
type Cinder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderSpec   `json:"spec,omitempty"`
	Status CinderStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CinderList contains a list of Cinder
type CinderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cinder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cinder{}, &CinderList{})
}
