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
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/storage/ceph"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DbSyncHash hash
	DbSyncHash = "dbsync"

	// DeploymentHash hash used to detect changes
	DeploymentHash = "deployment"
)

// CinderSpec defines the desired state of Cinder
type CinderSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=cinder
	// ServiceUser - optional username used for this service to register in cinder
	ServiceUser string `json:"serviceUser"`

	// +kubebuilder:validation:Required
	// MariaDB instance name
	// Right now required by the maridb-operator to get the credentials from the instance to create the DB
	// Might not be required in future
	DatabaseInstance string `json:"databaseInstance"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=cinder
	// DatabaseUser - optional username used for cinder DB, defaults to cinder
	// TODO: -> implement needs work in mariadb-operator, right now only cinder
	DatabaseUser string `json:"databaseUser"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=rabbitmq
	// RabbitMQ instance name
	// Needed to request a transportURL that is created and used in Cinder
	RabbitMqClusterName string `json:"rabbitMqClusterName"`

	// +kubebuilder:validation:Required
	// Secret containing OpenStack password information for CinderDatabasePassword, CinderPassword
	Secret string `json:"secret"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default={database: CinderDatabasePassword, service: CinderPassword}
	// PasswordSelectors - Selectors to identify the DB and ServiceUser password from the Secret
	PasswordSelectors PasswordSelector `json:"passwordSelectors,omitempty"`

	// +kubebuilder:validation:Optional
	// Debug - enable debug for different deploy stages. If an init container is used, it runs and the
	// actual action pod gets started with sleep infinity
	Debug CinderDebug `json:"debug,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// PreserveJobs - do not delete jobs after they finished e.g. to check logs
	PreserveJobs bool `json:"preserveJobs,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default="# add your customization here"
	// CustomServiceConfig - customize the service config for all Cinder services using this parameter to change service defaults,
	// or overwrite rendered information using raw OpenStack config format. The content gets added to
	// to /etc/<service>/<service>.conf.d directory as custom.conf file.
	CustomServiceConfig string `json:"customServiceConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ConfigOverwrite - interface to overwrite default config files like e.g. policy.json.
	// But can also be used to add additional files. Those get added to the service config dir in /etc/<service> .
	// TODO: -> implement
	DefaultConfigOverwrite map[string]string `json:"defaultConfigOverwrite,omitempty"`

	// TODO: We will need to decide which fields within the specs below we want to
	// be required or optional.  We probably also want to use webhooks and/or kubebuilder
	// defaults to set the fields when they are not provided by the user
	// TODO: As we flesh out functionality in the operator, we will need
	// to address these fields' optional/required/default considerations given
	// that CinderAPI, CinderBackup, CinderScheduler and CinderVolume are
	// intended to have their Specs embedded within this parent Cinder CRD.
	// Becuse we are embedding their specs, all admission field verification will
	// fire (as if an actual CR of one of the types had been applied/created).
	// We have to find the right balance between what we would expect users
	// to provide in the Cinder CR for the childrens' Specs versus what we
	// provide via inheriting from the Cinder CR or through webhooks, defaults, etc

	// +kubebuilder:validation:Required
	// CinderAPI - Spec definition for the API service of this Cinder deployment
	CinderAPI CinderAPISpec `json:"cinderAPI"`

	// +kubebuilder:validation:Required
	// CinderScheduler - Spec definition for the Scheduler service of this Cinder deployment
	CinderScheduler CinderSchedulerSpec `json:"cinderScheduler"`

	// +kubebuilder:validation:Optional
	// CinderBackup - Spec definition for the Backup service of this Cinder deployment
	CinderBackup CinderBackupSpec `json:"cinderBackup"`

	// +kubebuilder:validation:Optional
	// CinderVolumes - Map of chosen names to spec definitions for the Volume(s) service(s) of this Cinder deployment
	CinderVolumes map[string]CinderVolumeSpec `json:"cinderVolumes"`

	// +kubebuilder:validation:Optional
	// CephBackend - The ceph Backend structure with all the parameters
	CephBackend ceph.Backend `json:"cephBackend,omitempty"`
}

// CinderStatus defines the observed state of Cinder
type CinderStatus struct {
	// Map of hashes to track e.g. job status
	Hash map[string]string `json:"hash,omitempty"`

	// Conditions
	Conditions condition.Conditions `json:"conditions,omitempty" optional:"true"`

	// Cinder Database Hostname
	DatabaseHostname string `json:"databaseHostname,omitempty"`

	// TransportURLSecret - Secret containing RabbitMQ transportURL
	TransportURLSecret string `json:"transportURLSecret,omitempty"`

	// API endpoints
	APIEndpoints map[string]map[string]string `json:"apiEndpoints,omitempty"`

	// ServiceIDs
	ServiceIDs map[string]string `json:"serviceIDs,omitempty"`

	// ReadyCount of Cinder API instance
	CinderAPIReadyCount int32 `json:"cinderAPIReadyCount,omitempty"`

	// ReadyCount of Cinder Backup instance
	CinderBackupReadyCount int32 `json:"cinderBackupReadyCount,omitempty"`

	// ReadyCount of Cinder Scheduler instance
	CinderSchedulerReadyCount int32 `json:"cinderSchedulerReadyCount,omitempty"`

	// ReadyCounts of Cinder Volume instances
	CinderVolumesReadyCounts map[string]int32 `json:"cinderVolumesReadyCounts,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[0].status",description="Status"
//+kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.conditions[0].message",description="Message"

// Cinder is the Schema for the cinders API
type Cinder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CinderSpec   `json:"spec,omitempty"`
	Status CinderStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CinderList contains a list of Cinder
type CinderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cinder `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cinder{}, &CinderList{})
}

// IsReady - returns true if service is ready to serve requests
func (instance Cinder) IsReady() bool {
	ready := instance.Status.CinderAPIReadyCount > 0 && instance.Status.CinderBackupReadyCount > 0 &&
		instance.Status.CinderSchedulerReadyCount > 0

	for name := range instance.Spec.CinderVolumes {
		ready = ready && instance.Status.CinderVolumesReadyCounts[name] > 0
	}

	return ready
}
