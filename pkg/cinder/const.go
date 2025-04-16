/*

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

// Package cinder contains cinder service constants and configuration.
package cinder

import (
	"time"

	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/openstack-k8s-operators/lib-common/modules/storage"
)

const (
	// ServiceName -
	ServiceName = "cinder"
	// ServiceNameV3 -
	ServiceNameV3 = "cinderv3"
	// ServiceType -
	ServiceType = "cinder"
	// ServiceTypeV3 -
	ServiceTypeV3 = "volumev3"
	// DatabaseName -
	DatabaseName = "cinder"

	// DefaultsConfigFileName -
	DefaultsConfigFileName = "00-global-defaults.conf"
	// ServiceConfigFileName -
	ServiceConfigFileName = "01-service-defaults.conf"
	// CustomConfigFileName -
	CustomConfigFileName = "02-global-custom.conf"
	// CustomServiceConfigFileName -
	CustomServiceConfigFileName = "03-service-custom.conf"
	// CustomServiceConfigSecretsFileName -
	CustomServiceConfigSecretsFileName = "04-service-custom-secrets.conf"
	// MyCnfFileName -
	MyCnfFileName = "my.cnf"

	// CinderPublicPort -
	CinderPublicPort int32 = 8776
	// CinderInternalPort -
	CinderInternalPort int32 = 8776

	// CinderExtraVolTypeUndefined can be used to label an extraMount which
	// is not associated with a specific backend
	CinderExtraVolTypeUndefined storage.ExtraVolType = "Undefined"
	// CinderExtraVolTypeCeph can be used to label an extraMount which
	// is associated to a Ceph backend
	CinderExtraVolTypeCeph storage.ExtraVolType = "Ceph"
	// CinderBackup is the definition of the cinder-backup group
	CinderBackup storage.PropagationType = "CinderBackup"
	// CinderVolume is the definition of the cinder-volume group
	CinderVolume storage.PropagationType = "CinderVolume"
	// CinderScheduler is the definition of the cinder-scheduler group
	CinderScheduler storage.PropagationType = "CinderScheduler"
	// CinderAPI is the definition of the cinder-api group
	CinderAPI storage.PropagationType = "CinderAPI"
	// Cinder is the global ServiceType that refers to all the components deployed
	// by the cinder operator
	Cinder storage.PropagationType = "Cinder"

	// ShortDuration is a short duration for quick retries
	ShortDuration = time.Duration(5) * time.Second
	// NormalDuration is the normal duration for standard retries
	NormalDuration = time.Duration(10) * time.Second
)

// ResultRequeue is the standard requeue result with normal duration
var ResultRequeue = ctrl.Result{RequeueAfter: NormalDuration}

// DbsyncPropagation keeps track of the DBSync Service Propagation Type
var DbsyncPropagation = []storage.PropagationType{storage.DBSync}

// CinderSchedulerPropagation is the  definition of the CinderScheduler propagation group
// It allows the CinderScheduler pod to mount volumes destined to Cinder and CinderScheduler
// ServiceTypes
var CinderSchedulerPropagation = []storage.PropagationType{Cinder, CinderScheduler}

// CinderAPIPropagation is the  definition of the CinderAPI propagation group
// It allows the CinderAPI pod to mount volumes destined to Cinder and CinderAPI
// ServiceTypes
var CinderAPIPropagation = []storage.PropagationType{Cinder, CinderAPI}

// CinderBackupPropagation is the  definition of the CinderBackup propagation group
// It allows the CinderBackup pod to mount volumes destined to Cinder and CinderBackup
// ServiceTypes
var CinderBackupPropagation = []storage.PropagationType{Cinder, CinderBackup}

// CinderVolumePropagation is the  definition of the CinderVolume propagation group
// It allows the CinderVolume pods to mount volumes destined to Cinder and CinderVolume
// ServiceTypes
var CinderVolumePropagation = []storage.PropagationType{Cinder, CinderVolume}
