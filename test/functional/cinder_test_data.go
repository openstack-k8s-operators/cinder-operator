/*
Copyright 2023.

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

// Package functional implements the envTest coverage for cinder-operator
package functional

import (
	"fmt"

	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// MemcachedInstance - name of the memcached instance
	MemcachedInstance = "memcached"
	//PublicCertSecretName -
	PublicCertSecretName = "public-tls-certs"
	//InternalCertSecretName -
	InternalCertSecretName = "internal-tls-certs"
	//CABundleSecretName -
	CABundleSecretName = "combined-ca-bundle"
	// CinderCephExtraMountsPath -
	CinderCephExtraMountsPath = "/etc/ceph"
	// CinderCephExtraMountsSecretName -
	CinderCephExtraMountsSecretName = "ceph"
)

// CinderTestData is the data structure used to provide input data to envTest
type CinderTestData struct {
	RabbitmqClusterName            string
	RabbitmqSecretName             string
	MemcachedInstance              string
	NotificationSecretName         string
	CinderDataBaseAccount          string
	CinderPassword                 string
	CinderServiceUser              string
	DatabaseHostname               string
	Instance                       types.NamespacedName
	CinderRole                     types.NamespacedName
	CinderRoleBinding              types.NamespacedName
	CinderTransportURL             types.NamespacedName
	CinderTransportURLNotification types.NamespacedName
	CinderMemcached                types.NamespacedName
	CinderSA                       types.NamespacedName
	CinderDBSync                   types.NamespacedName
	CinderDBPurge                  types.NamespacedName
	CinderKeystoneService          types.NamespacedName
	CinderKeystoneEndpoint         types.NamespacedName
	CinderServicePublic            types.NamespacedName
	CinderServiceInternal          types.NamespacedName
	CinderConfigSecret             types.NamespacedName
	CinderConfigScripts            types.NamespacedName
	Cinder                         types.NamespacedName
	CinderAPI                      types.NamespacedName
	CinderScheduler                types.NamespacedName
	CinderVolumes                  []types.NamespacedName
	InternalAPINAD                 types.NamespacedName
	ContainerImage                 string
	CABundleSecret                 types.NamespacedName
	InternalCertSecret             types.NamespacedName
	PublicCertSecret               types.NamespacedName
	Database                       types.NamespacedName
	CinderTopologies               []types.NamespacedName
}

// GetCinderTestData is a function that initialize the CinderTestData
// used in the test
func GetCinderTestData(cinderName types.NamespacedName) CinderTestData {

	m := cinderName
	return CinderTestData{
		Instance: m,

		Cinder: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      cinderName.Name,
		},
		CinderDBSync: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-db-sync", cinderName.Name),
		},
		CinderDBPurge: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      "cinder-db-purge",
		},
		CinderAPI: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-api", cinderName.Name),
		},
		CinderScheduler: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-scheduler", cinderName.Name),
		},
		CinderVolumes: []types.NamespacedName{
			{
				Namespace: cinderName.Namespace,
				Name:      fmt.Sprintf("%s-volume-volume1", cinderName.Name),
			},
			{
				Namespace: cinderName.Namespace,
				Name:      fmt.Sprintf("%s-volume-volume2", cinderName.Name),
			},
		},
		CinderRole: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("cinder-%s-role", cinderName.Name),
		},
		CinderRoleBinding: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("cinder-%s-rolebinding", cinderName.Name),
		},
		CinderSA: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("cinder-%s", cinderName.Name),
		},
		CinderTransportURL: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-cinder-transport", cinderName.Name),
		},
		CinderTransportURLNotification: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-cinder-transport-notification", cinderName.Name),
		},
		CinderMemcached: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      MemcachedInstance,
		},
		CinderConfigSecret: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-%s", cinderName.Name, "config-data"),
		},
		CinderConfigScripts: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-%s", cinderName.Name, "scripts"),
		},
		// Also used to identify CinderRoutePublic
		CinderServicePublic: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-public", cinder.ServiceName),
		},
		CinderServiceInternal: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-internal", cinder.ServiceName),
		},
		CinderKeystoneService: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      cinder.ServiceNameV3,
		},
		CinderKeystoneEndpoint: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      cinder.ServiceNameV3,
		},
		InternalAPINAD: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      "internalapi",
		},
		RabbitmqClusterName:    "rabbitmq",
		RabbitmqSecretName:     "rabbitmq-secret",
		NotificationSecretName: "rabbitmq-notifications-secret",
		MemcachedInstance:      MemcachedInstance,
		CinderDataBaseAccount:  "cinder",
		// Password used for both db and service
		CinderPassword:    "12345678",
		CinderServiceUser: "cinder",
		ContainerImage:    "test://cinder",
		DatabaseHostname:  "database-hostname",
		CABundleSecret: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      CABundleSecretName,
		},
		InternalCertSecret: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      InternalCertSecretName,
		},
		PublicCertSecret: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      PublicCertSecretName,
		},
		Database: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      cinder.DatabaseName,
		},
		// A set of topologies to Test how the reference is propagated to the
		// resulting StatefulSets and if a potential override produces the
		// expected values
		CinderTopologies: []types.NamespacedName{
			{
				Namespace: cinderName.Namespace,
				Name:      fmt.Sprintf("%s-global-topology", cinderName.Name),
			},
			{
				Namespace: cinderName.Namespace,
				Name:      fmt.Sprintf("%s-api-topology", cinderName.Name),
			},
			{
				Namespace: cinderName.Namespace,
				Name:      fmt.Sprintf("%s-scheduler-topology", cinderName.Name),
			},
			{
				Namespace: cinderName.Namespace,
				Name:      fmt.Sprintf("%s-volume-topology", cinderName.Name),
			},
		},
	}
}
