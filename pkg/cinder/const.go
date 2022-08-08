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

package cinder

const (
	// ServiceName -
	ServiceName = "cinder"
	// ServiceNameV2 -
	ServiceNameV2 = "cinderv2"
	// ServiceNameV3 -
	ServiceNameV3 = "cinderv3"
	// ServiceType -
	ServiceType = "cinder"
	// ServiceAccount -
	ServiceAccount = "cinder-operator-cinder"
	// ServiceTypeV2 -
	ServiceTypeV2 = "volumev2"
	// ServiceTypeV3 -
	ServiceTypeV3 = "volumev3"
	// DatabaseName -
	DatabaseName = "cinder"

	// CinderAdminPort -
	CinderAdminPort int32 = 8776
	// CinderPublicPort -
	CinderPublicPort int32 = 8776
	// CinderInternalPort -
	CinderInternalPort int32 = 8776

	// KollaConfigDbSync -
	KollaConfigDbSync = "/var/lib/config-data/merged/db-sync-config.json"
)
