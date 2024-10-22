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
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CinderUserID - Kolla's cinder UID comes from the 'cinder-user' in
	// https://github.com/openstack/kolla/blob/master/kolla/common/users.py
	CinderUserID = 42407
	// CinderGroupID - Kolla's cinder GID
	CinderGroupID = 42407

	// DbSyncHash hash
	DbSyncHash = "dbsync"

	// DeploymentHash hash used to detect changes
	DeploymentHash = "deployment"

	// Container image fall-back defaults

	// CinderAPIContainerImage is the fall-back container image for CinderAPI
	CinderAPIContainerImage = "quay.io/podified-antelope-centos9/openstack-cinder-api:current-podified"
	// CinderBackupContainerImage is the fall-back container image for CinderBackup
	CinderBackupContainerImage = "quay.io/podified-antelope-centos9/openstack-cinder-backup:current-podified"
	// CinderSchedulerContainerImage is the fall-back container image for CinderScheduler
	CinderSchedulerContainerImage = "quay.io/podified-antelope-centos9/openstack-cinder-scheduler:current-podified"
	// CinderVolumeContainerImage is the fall-back container image for CinderVolume
	CinderVolumeContainerImage = "quay.io/podified-antelope-centos9/openstack-cinder-volume:current-podified"

	// DBPurgeDefaultAge - Default age, in days, for purging deleted DB records
	DBPurgeDefaultAge = 30
	// DBPurgeDefaultSchedule - Default cron schedule for purging the DB
	DBPurgeDefaultSchedule = "1 0 * * *"
	// APITimeoutDefault  - Default timeout in seconds for HAProxy, Apache, and RPCs
	APITimeoutDefault = 60
)

type CinderSpecBase struct {
	CinderTemplate `json:",inline"`

	// +kubebuilder:validation:Required
	// MariaDB instance name
	// Right now required by the maridb-operator to get the credentials from the instance to create the DB
	// Might not be required in future
	DatabaseInstance string `json:"databaseInstance"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=rabbitmq
	// RabbitMQ instance name
	// Needed to request a transportURL that is created and used in Cinder
	RabbitMqClusterName string `json:"rabbitMqClusterName"`

	// +kubebuilder:validation:Required
	// +kubebuilder:default=memcached
	// Memcached instance name.
	MemcachedInstance string `json:"memcachedInstance"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// PreserveJobs - do not delete jobs after they finished e.g. to check logs
	PreserveJobs bool `json:"preserveJobs"`

	// +kubebuilder:validation:Optional
	// CustomServiceConfig - customize the service config for all Cinder services using this parameter to change service defaults,
	// or overwrite rendered information using raw OpenStack config format. The content gets added to
	// to /etc/<service>/<service>.conf.d directory as a custom config file.
	CustomServiceConfig string `json:"customServiceConfig,omitempty"`

	// +kubebuilder:validation:Optional
	// ExtraMounts containing conf files and credentials
	ExtraMounts []CinderExtraVolMounts `json:"extraMounts,omitempty"`

	// +kubebuilder:validation:Optional
	// NodeSelector to target subset of worker nodes running this service. Setting
	// NodeSelector here acts as a default value and can be overridden by service
	// specific NodeSelector Settings.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	// DBPurge parameters -
	DBPurge DBPurge `json:"dbPurge,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:default=60
	// +kubebuilder:validation:Minimum=10
	// APITimeout for HAProxy, Apache, and rpc_response_timeout
	APITimeout int `json:"apiTimeout"`
}

// CinderSpecCore the same as CinderSpec without ContainerImage references
type CinderSpecCore struct {
	CinderSpecBase `json:",inline"`

	// CinderAPI - Spec definition for the API service of this Cinder deployment
	CinderAPI CinderAPITemplateCore `json:"cinderAPI"`

	// +kubebuilder:validation:Required
	// CinderScheduler - Spec definition for the Scheduler service of this Cinder deployment
	CinderScheduler CinderSchedulerTemplateCore `json:"cinderScheduler"`

	// +kubebuilder:validation:Optional
	// CinderBackup - Spec definition for the Backup service of this Cinder deployment
	CinderBackup CinderBackupTemplateCore `json:"cinderBackup"`

	// +kubebuilder:validation:Optional
	// CinderVolumes - Map of chosen names to spec definitions for the Volume(s) service(s) of this Cinder deployment
	CinderVolumes map[string]CinderVolumeTemplateCore `json:"cinderVolumes,omitempty"`
}

// CinderSpec defines the desired state of Cinder
type CinderSpec struct {
	CinderSpecBase `json:",inline"`

	// +kubebuilder:validation:Required
	// CinderAPI - Spec definition for the API service of this Cinder deployment
	CinderAPI CinderAPITemplate `json:"cinderAPI"`

	// +kubebuilder:validation:Required
	// CinderScheduler - Spec definition for the Scheduler service of this Cinder deployment
	CinderScheduler CinderSchedulerTemplate `json:"cinderScheduler"`

	// +kubebuilder:validation:Optional
	// CinderBackup - Spec definition for the Backup service of this Cinder deployment
	CinderBackup CinderBackupTemplate `json:"cinderBackup"`

	// +kubebuilder:validation:Optional
	// CinderVolumes - Map of chosen names to spec definitions for the Volume(s) service(s) of this Cinder deployment
	CinderVolumes map[string]CinderVolumeTemplate `json:"cinderVolumes,omitempty"`
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
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	CinderAPIReadyCount int32 `json:"cinderAPIReadyCount"`

	// ReadyCount of Cinder Backup instance
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	CinderBackupReadyCount int32 `json:"cinderBackupReadyCount"`

	// ReadyCount of Cinder Scheduler instance
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=0
	CinderSchedulerReadyCount int32 `json:"cinderSchedulerReadyCount"`

	// ReadyCounts of Cinder Volume instances
	CinderVolumesReadyCounts map[string]int32 `json:"cinderVolumesReadyCounts,omitempty"`

	// ObservedGeneration - the most recent generation observed for this service.
	// If the observed generation is different than the spec generation, then the
	// controller has not started processing the latest changes, and the status
	// and its conditions are likely stale.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
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

// DBPurge struct is used to model the parameters exposed to the Cinder cronJob
type DBPurge struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=1
	// Age is the DBPurgeAge parameter and indicates the number of days of purging DB records
	Age int `json:"age"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="1 0 * * *"
	// Schedule defines the crontab format string to schedule the DBPurge cronJob
	Schedule string `json:"schedule"`
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

// IsReady - returns true if all subresources Ready condition is true
func (instance Cinder) IsReady() bool {
	return instance.Generation == instance.Status.ObservedGeneration &&
		instance.Status.Conditions.IsTrue(CinderAPIReadyCondition) &&
		instance.Status.Conditions.IsTrue(CinderBackupReadyCondition) &&
		instance.Status.Conditions.IsTrue(CinderSchedulerReadyCondition) &&
		instance.Status.Conditions.IsTrue(CinderVolumeReadyCondition)
}

// CinderExtraVolMounts exposes additional parameters processed by the cinder-operator
// and defines the common VolMounts structure provided by the main storage module
type CinderExtraVolMounts struct {
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
	// +kubebuilder:validation:Optional
	Region string `json:"region,omitempty"`
	// +kubebuilder:validation:Required
	VolMounts []storage.VolMounts `json:"extraVol"`
}

// Propagate is a function used to filter VolMounts according to the specified
// PropagationType array
func (c *CinderExtraVolMounts) Propagate(svc []storage.PropagationType) []storage.VolMounts {

	var vl []storage.VolMounts

	for _, gv := range c.VolMounts {
		vl = append(vl, gv.Propagate(svc)...)
	}

	return vl
}

// RbacConditionsSet - set the conditions for the rbac object
func (instance Cinder) RbacConditionsSet(c *condition.Condition) {
	instance.Status.Conditions.Set(c)
}

// RbacNamespace - return the namespace
func (instance Cinder) RbacNamespace() string {
	return instance.Namespace
}

// RbacResourceName - return the name to be used for rbac objects (serviceaccount, role, rolebinding)
func (instance Cinder) RbacResourceName() string {
	return "cinder-" + instance.Name
}
