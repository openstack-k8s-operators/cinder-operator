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

package functional

import (
	"fmt"

	"golang.org/x/exp/maps"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cinderv1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
)

func CreateCinderSecret(namespace string, name string) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: name},
		map[string][]byte{
			"CinderPassword":         []byte(cinderTest.CinderPassword),
			"CinderDatabasePassword": []byte(cinderTest.CinderPassword),
			"MetadataSecret":         []byte(cinderTest.CinderPassword),
		},
	)
}

func CreateCinderMessageBusSecret(namespace string, name string) *corev1.Secret {
	s := th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: name},
		map[string][]byte{
			"transport_url": []byte(fmt.Sprintf("rabbit://%s/fake", name)),
		},
	)
	logger.Info("Secret created", "name", name)
	return s
}

func CreateUnstructured(rawObj map[string]interface{}) *unstructured.Unstructured {
	logger.Info("Creating", "raw", rawObj)
	unstructuredObj := &unstructured.Unstructured{Object: rawObj}
	_, err := controllerutil.CreateOrPatch(
		ctx, k8sClient, unstructuredObj, func() error { return nil })
	Expect(err).ShouldNot(HaveOccurred())
	return unstructuredObj
}

func GetCinderEmptySpec() map[string]interface{} {
	return map[string]interface{}{
		"databaseInstance": "openstack",
		"secret":           SecretName,
	}
}

func GetDefaultCinderSpec() map[string]interface{} {
	return map[string]interface{}{
		"databaseInstance": "openstack",
		"secret":           SecretName,
		"cinderAPI":        GetDefaultCinderAPISpec(),
		"cinderScheduler":  GetDefaultCinderSchedulerSpec(),
		"cinderVolume":     GetDefaultCinderVolumeSpec(),
	}
}

func GetTLSCinderSpec() map[string]interface{} {
	return map[string]interface{}{
		"databaseInstance": "openstack",
		"secret":           SecretName,
		"cinderAPI":        GetTLSCinderAPISpec(),
		"cinderScheduler":  GetDefaultCinderSchedulerSpec(),
		"cinderVolume":     GetDefaultCinderVolumeSpec(),
	}
}

func GetDefaultCinderAPISpec() map[string]interface{} {
	return map[string]interface{}{
		"secret":             SecretName,
		"replicas":           1,
		"containerImage":     cinderTest.ContainerImage,
		"serviceAccount":     cinderTest.CinderSA.Name,
		"databaseHostname":   cinderTest.DatabaseHostname,
		"transportURLSecret": cinderTest.RabbitmqSecretName,
	}
}

func GetTLSCinderAPISpec() map[string]interface{} {
	spec := GetDefaultCinderAPISpec()
	maps.Copy(spec, map[string]interface{}{
		"tls": map[string]interface{}{
			"api": map[string]interface{}{
				"internal": map[string]interface{}{
					"secretName": InternalCertSecretName,
				},
				"public": map[string]interface{}{
					"secretName": PublicCertSecretName,
				},
			},
			"caBundleSecretName": CABundleSecretName,
		},
	})

	return spec
}

func GetDefaultCinderSchedulerSpec() map[string]interface{} {
	return map[string]interface{}{
		"secret":             SecretName,
		"replicas":           1,
		"containerImage":     cinderTest.ContainerImage,
		"serviceAccount":     cinderTest.CinderSA.Name,
		"databaseHostname":   cinderTest.DatabaseHostname,
		"transportURLSecret": cinderTest.RabbitmqSecretName,
	}
}

func GetDefaultCinderVolumeSpec() map[string]interface{} {
	return map[string]interface{}{
		"secret":             SecretName,
		"replicas":           1,
		"containerImage":     cinderTest.ContainerImage,
		"serviceAccount":     cinderTest.CinderSA.Name,
		"databaseHostname":   cinderTest.DatabaseHostname,
		"transportURLSecret": cinderTest.RabbitmqSecretName,
	}
}

func GetCinder(name types.NamespacedName) *cinderv1.Cinder {
	instance := &cinderv1.Cinder{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func CreateCinder(name types.NamespacedName, spec map[string]interface{}) client.Object {

	raw := map[string]interface{}{
		"apiVersion": "cinder.openstack.org/v1beta1",
		"kind":       "Cinder",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func CinderConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetCinder(name)
	return instance.Status.Conditions
}

func CreateCinderAPI(name types.NamespacedName, spec map[string]interface{}) client.Object {
	raw := map[string]interface{}{
		"apiVersion": "cinder.openstack.org/v1beta1",
		"kind":       "CinderAPI",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func CreateCinderScheduler(name types.NamespacedName, spec map[string]interface{}) client.Object {
	raw := map[string]interface{}{
		"apiVersion": "cinder.openstack.org/v1beta1",
		"kind":       "CinderScheduler",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func CreateCinderVolume(name types.NamespacedName, spec map[string]interface{}) client.Object {
	raw := map[string]interface{}{
		"apiVersion": "cinder.openstack.org/v1beta1",
		"kind":       "CinderVolume",
		"metadata": map[string]interface{}{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func GetCinderAPI(name types.NamespacedName) *cinderv1.CinderAPI {
	instance := &cinderv1.CinderAPI{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func GetCinderScheduler(name types.NamespacedName) *cinderv1.CinderScheduler {
	instance := &cinderv1.CinderScheduler{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func GetCinderVolume(name types.NamespacedName) *cinderv1.CinderVolume {
	instance := &cinderv1.CinderVolume{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func CinderAPIConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetCinderAPI(name)
	return instance.Status.Conditions
}

func CinderSchedulerConditionGetter(name types.NamespacedName) condition.Conditions {
	instance := GetCinderScheduler(name)
	return instance.Status.Conditions
}

func CinderAPINotExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &cinderv1.CinderAPI{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

func CinderAPIExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &cinderv1.CinderAPI{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeFalse())
	}, timeout, interval).Should(Succeed())
}

func CinderSchedulerExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &cinderv1.CinderScheduler{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeFalse())
	}, timeout, interval).Should(Succeed())
}

func CinderVolumeExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &cinderv1.CinderVolume{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeFalse())
	}, timeout, interval).Should(Succeed())
}

func CinderSchedulerNotExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &cinderv1.CinderScheduler{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}

func CinderVolumeNotExists(name types.NamespacedName) {
	Consistently(func(g Gomega) {
		instance := &cinderv1.CinderVolume{}
		err := k8sClient.Get(ctx, name, instance)
		g.Expect(k8s_errors.IsNotFound(err)).To(BeTrue())
	}, timeout, interval).Should(Succeed())
}
