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

package v1beta1

import (
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

//
// Cinder Condition Types used by API objects.
//
const (
	// CinderAPIReadyCondition Status=True condition which indicates if the CinderAPI is configured and operational
	CinderAPIReadyCondition condition.Type = "CinderAPIReady"

	// CinderSchedulerReadyCondition Status=True condition which indicates if the CinderScheduler is configured and operational
	CinderSchedulerReadyCondition condition.Type = "CinderSchedulerReady"

	// CinderBackupReadyCondition Status=True condition which indicates if the CinderBackup is configured and operational
	CinderBackupReadyCondition condition.Type = "CinderBackupReady"

	// CinderVolumeReadyCondition Status=True condition which indicates if the CinderVolume is configured and operational
	CinderVolumeReadyCondition condition.Type = "CinderVolumeReady"
)

//
// Cinder Reasons used by API objects.
//
const ()

//
// Common Messages used by API objects.
//
const (
	//
	// CinderAPIReady condition messages
	//
	// CinderAPIReadyInitMessage
	CinderAPIReadyInitMessage = "CinderAPI not started"

	// CinderAPIReadyErrorMessage
	CinderAPIReadyErrorMessage = "CinderAPI error occured %s"

	//
	// CinderSchedulerReady condition messages
	//
	// CinderSchedulerReadyInitMessage
	CinderSchedulerReadyInitMessage = "CinderScheduler not started"

	// CinderSchedulerReadyErrorMessage
	CinderSchedulerReadyErrorMessage = "CinderScheduler error occured %s"

	//
	// CinderBackupReady condition messages
	//
	// CinderBackupReadyInitMessage
	CinderBackupReadyInitMessage = "CinderBackup not started"

	// CinderBackupReadyErrorMessage
	CinderBackupReadyErrorMessage = "CinderBackup error occured %s"

	//
	// CinderVolumeReady condition messages
	//
	// CinderVolumeReadyInitMessage
	CinderVolumeReadyInitMessage = "CinderVolume not started"

	// CinderVolumeReadyErrorMessage
	CinderVolumeReadyErrorMessage = "CinderVolume error occured %s"

	// CinderVolumeReadyRunningMessage
	CinderVolumeReadyRunningMessage = "CinderVolume deployments in progress"
)
