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

import (
	"fmt"
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"net"
	"sort"
	"strings"
)

// CephDefaults is used as type to reference defaults
type CephDefaults string

// Spec is the Cinder Ceph Backend struct defining defaults
type Spec struct {
	CephDefaults *CephDefaults
}

// Cinder defaults
const (
	CephDefaultCinderUser CephDefaults = "openstack"
	CephDefaultCinderPool CephDefaults = "volumes"
)

// GetCephBackend is a function that validates the ceph client parameters.
func GetCephBackend(instance *cinderv1beta1.Cinder) bool {
	// Validate parameters to enable CephBackend
	if instance.Spec.CephBackend.CephClusterFSID != "" &&
		validateMons(instance.Spec.CephBackend.CephClusterMonHosts) &&
		instance.Spec.CephBackend.CephClientKey != "" {
		return true
	}
	return false
}

// This is a utily function that validates the comma separated Mon list defined
// for the external ceph cluster; it also checks the provided IP addresses are not
// malformed
func validateMons(ipList string) bool {
	for _, ip := range strings.Split(ipList, ",") {
		if net.ParseIP(strings.Trim(ip, " ")) == nil {
			return false
		}
	}
	return true
}

// GetCephCinderPool is a function that validate the pool passed as input, and return
// volumes no pool is given
func GetCephCinderPool(instance *cinderv1beta1.Cinder) string {

	if pool, found := instance.Spec.CephBackend.CephPools["cinder"]; found {
		return pool.CephPoolName
	}
	return string(CephDefaultCinderPool)
}

// GetCephRbdUser is a function that validate the user passed as input, and return
// openstack if no user is given
func GetCephRbdUser(instance *cinderv1beta1.Cinder) string {

	if instance.Spec.CephBackend.CephUser == "" {
		return string(CephDefaultCinderUser)
	}
	return instance.Spec.CephBackend.CephUser
}

// GetCephOsdCaps is a function that returns the Caps for each defined pool
func GetCephOsdCaps(instance *cinderv1beta1.Cinder) string {

	var osdCaps string // the resulting string containing caps

	/**
	A map of strings (pool service/name in this case) is, by definition, an
	unordered structure, and let the function return a different pattern
	each time. This causes the ConfigMap hash to change, and the pod being
	redeployed because the operator detects the different hash. Sorting the
	resulting array of pools makes everything predictable
	**/
	var plist []string
	for _, pool := range instance.Spec.CephBackend.CephPools {
		plist = append(plist, pool.CephPoolName)
	}
	// sort the pool names
	sort.Strings(plist)

	// Build the resulting caps from the _ordered_ array applying the template
	for _, pool := range plist {
		if pool != "" {
			osdCaps += fmt.Sprintf("profile rbd pool=%s,", pool)
		}
	}
	// Default case, no pools are specified, adding just "volumes" (the default)
	if osdCaps == "" {
		osdCaps = "profile rbd pool=" + string(CephDefaultCinderPool) + ","
	}

	return strings.TrimSuffix(osdCaps, ",")
}
