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
	"k8s.io/apimachinery/pkg/types"
)

// CinderTestData is the data structure used to provide input data to envTest
type CinderTestData struct {
	RabbitmqClusterName    string
	RabbitmqSecretName     string
	CinderDataBaseUser     string
	CinderPassword         string
	CinderServiceUser      string
	Instance               types.NamespacedName
	CinderRole             types.NamespacedName
	CinderRoleBinding      types.NamespacedName
	CinderTransportURL     types.NamespacedName
	CinderSA               types.NamespacedName
	CinderDBSync           types.NamespacedName
	CinderKeystoneEndpoint types.NamespacedName
	CinderServicePublic    types.NamespacedName
	CinderServiceInternal  types.NamespacedName
	CinderConfigSecret     types.NamespacedName
	CinderConfigScripts    types.NamespacedName
	CinderAPI              types.NamespacedName
	CinderScheduler        types.NamespacedName
	CinderVolumes          []types.NamespacedName
	InternalAPINAD         types.NamespacedName
	ContainerImage         string
}

// GetCinderTestData is a function that initialize the CinderTestData
// used in the test
func GetCinderTestData(cinderName types.NamespacedName) CinderTestData {

	m := cinderName
	return CinderTestData{
		Instance: m,

		CinderDBSync: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-db-sync", cinderName.Name),
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
			Name:      fmt.Sprintf("cinder-%s-transport", cinderName.Name),
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
			Name:      fmt.Sprintf("%s-public", cinderName.Name),
		},
		// Also used to identify CinderKeystoneService
		CinderServiceInternal: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%s-internal", cinderName.Name),
		},
		CinderKeystoneEndpoint: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      fmt.Sprintf("%sv3", cinderName.Name),
		},
		InternalAPINAD: types.NamespacedName{
			Namespace: cinderName.Namespace,
			Name:      "internalapi",
		},
		RabbitmqClusterName: "rabbitmq",
		RabbitmqSecretName:  "rabbitmq-secret",
		CinderDataBaseUser:  "cinder",
		// Password used for both db and service
		CinderPassword:    "12345678",
		CinderServiceUser: "cinder",
		ContainerImage:    "test://cinder",
	}
}
