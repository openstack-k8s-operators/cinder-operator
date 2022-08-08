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

// PasswordSelector to identify the DB and AdminUser password from the Secret
type PasswordSelector struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="CinderDatabasePassword"
	// Database - Selector to get the cinder database user password from the Secret
	// TODO: not used, need change in mariadb-operator
	Database string `json:"database,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="CinderPassword"
	// Database - Selector to get the cinder service password from the Secret
	Service string `json:"admin,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="TransportURL"
	// Database - Selector to get the cinder service password from the Secret
	TransportURL string `json:"transportUrl,omitempty"`
}

// CinderDebug indicates whether certain stages of Cinder deployment should pause in debug mode
type CinderDebug struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// DBSync enable debug
	DBSync bool `json:"dbSync,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Bootstrap enable debug
	Bootstrap bool `json:"bootstrap,omitempty"`
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	// Service enable debug
	Service bool `json:"service,omitempty"`
}
