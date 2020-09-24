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

package common

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	util "github.com/openstack-k8s-operators/lib-common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetSecretsFromCR - get all parameters ending with "Secret" from Spec to verify they exist, add the hash to env and status
func GetSecretsFromCR(r ReconcilerCommon, obj runtime.Object, namespace string, spec interface{}, envVars *map[string]util.EnvSetter) ([]cinderv1beta1.Hash, error) {
	hashes := []cinderv1beta1.Hash{}
	specParameters := make(map[string]interface{})
	inrec, _ := json.Marshal(spec)
	json.Unmarshal(inrec, &specParameters)

	for param, value := range specParameters {
		if strings.HasSuffix(param, "Secret") {
			_, hash, err := GetSecret(r.GetClient(), fmt.Sprintf("%v", value), namespace)
			if err != nil {
				return nil, err
			}

			// add hash to envVars
			(*envVars)[param] = util.EnvValue(hash)
			hashes = append(hashes, cinderv1beta1.Hash{Name: param, Hash: hash})
		}
	}

	return hashes, nil
}

// GetSecret -
func GetSecret(c client.Client, secretName string, secretNamespace string) (*corev1.Secret, string, error) {
	secret := &corev1.Secret{}

	err := c.Get(context.TODO(), types.NamespacedName{Name: secretName, Namespace: secretNamespace}, secret)
	if err != nil {
		return nil, "", err
	}

	secretHash, err := util.ObjectHash(secret)
	if err != nil {
		return nil, "", fmt.Errorf("error calculating configuration hash: %v", err)
	}
	return secret, secretHash, nil
}
