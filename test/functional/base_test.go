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

	. "github.com/onsi/gomega" //revive:disable:dot-imports
	cinderv1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/internal/cinder"
	keystonev1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
			"transport_url": fmt.Appendf(nil, "rabbit://%s/fake", name),
		},
	)
	logger.Info("Secret created", "name", name)
	return s
}

func CreateUnstructured(rawObj map[string]any) *unstructured.Unstructured {
	logger.Info("Creating", "raw", rawObj)
	unstructuredObj := &unstructured.Unstructured{Object: rawObj}
	_, err := controllerutil.CreateOrPatch(
		ctx, k8sClient, unstructuredObj, func() error { return nil })
	Expect(err).ShouldNot(HaveOccurred())
	return unstructuredObj
}

func GetCinderEmptySpec() map[string]any {
	return map[string]any{
		"databaseInstance": "openstack",
		"secret":           SecretName,
	}
}

func GetDefaultCinderSpec() map[string]any {
	return map[string]any{
		"databaseInstance": "openstack",
		"secret":           SecretName,
		"cinderAPI":        GetDefaultCinderAPISpec(),
		"cinderScheduler":  GetDefaultCinderSchedulerSpec(),
		"cinderVolume":     GetDefaultCinderVolumeSpec(),
	}
}

func GetTLSCinderSpec() map[string]any {
	return map[string]any{
		"databaseInstance": "openstack",
		"secret":           SecretName,
		"cinderAPI":        GetTLSCinderAPISpec(),
		"cinderScheduler":  GetDefaultCinderSchedulerSpec(),
		"cinderVolume":     GetDefaultCinderVolumeSpec(),
	}
}

func GetDefaultCinderAPISpec() map[string]any {
	return map[string]any{
		"secret":             SecretName,
		"replicas":           1,
		"containerImage":     cinderTest.ContainerImage,
		"serviceAccount":     cinderTest.CinderSA.Name,
		"databaseHostname":   cinderTest.DatabaseHostname,
		"transportURLSecret": cinderTest.RabbitmqSecretName,
	}
}

func GetTLSCinderAPISpec() map[string]any {
	spec := GetDefaultCinderAPISpec()
	maps.Copy(spec, map[string]any{
		"tls": map[string]any{
			"api": map[string]any{
				"internal": map[string]any{
					"secretName": InternalCertSecretName,
				},
				"public": map[string]any{
					"secretName": PublicCertSecretName,
				},
			},
			"caBundleSecretName": CABundleSecretName,
		},
	})

	return spec
}

func GetDefaultCinderSchedulerSpec() map[string]any {
	return map[string]any{
		"secret":             SecretName,
		"replicas":           1,
		"containerImage":     cinderTest.ContainerImage,
		"serviceAccount":     cinderTest.CinderSA.Name,
		"databaseHostname":   cinderTest.DatabaseHostname,
		"transportURLSecret": cinderTest.RabbitmqSecretName,
	}
}

func GetDefaultCinderVolumeSpec() map[string]any {
	return map[string]any{
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

func CreateCinder(name types.NamespacedName, spec map[string]any) client.Object {

	raw := map[string]any{
		"apiVersion": "cinder.openstack.org/v1beta1",
		"kind":       "Cinder",
		"metadata": map[string]any{
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

func CreateCinderAPI(name types.NamespacedName, spec map[string]any) client.Object {
	raw := map[string]any{
		"apiVersion": "cinder.openstack.org/v1beta1",
		"kind":       "CinderAPI",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func CreateCinderScheduler(name types.NamespacedName, spec map[string]any) client.Object {
	raw := map[string]any{
		"apiVersion": "cinder.openstack.org/v1beta1",
		"kind":       "CinderScheduler",
		"metadata": map[string]any{
			"name":      name.Name,
			"namespace": name.Namespace,
		},
		"spec": spec,
	}
	return CreateUnstructured(raw)
}

func CreateCinderVolume(name types.NamespacedName, spec map[string]any) client.Object {
	raw := map[string]any{
		"apiVersion": "cinder.openstack.org/v1beta1",
		"kind":       "CinderVolume",
		"metadata": map[string]any{
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

func GetCinderAPISpec(name types.NamespacedName) cinderv1.CinderAPITemplate {
	instance := &cinderv1.CinderAPI{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance.Spec.CinderAPITemplate
}

func GetCinderSchedulerSpec(name types.NamespacedName) cinderv1.CinderSchedulerTemplate {
	instance := &cinderv1.CinderScheduler{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance.Spec.CinderSchedulerTemplate
}

func GetCinderVolumeSpec(name types.NamespacedName) cinderv1.CinderVolumeTemplate {
	instance := &cinderv1.CinderVolume{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance.Spec.CinderVolumeTemplate
}

func GetCinderVolume(name types.NamespacedName) *cinderv1.CinderVolume {
	instance := &cinderv1.CinderVolume{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

func GetCronJob(name types.NamespacedName) *batchv1.CronJob {
	instance := &batchv1.CronJob{}
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

// GetExtraMounts - Utility function that simulates extraMounts pointing
// to a Ceph secret
func GetExtraMounts() []map[string]any {
	return []map[string]any{
		{
			"name":   cinderTest.Instance.Name,
			"region": "az0",
			"extraVol": []map[string]any{
				{
					"extraVolType": CinderCephExtraMountsSecretName,
					"propagation": []string{
						"CinderVolume",
					},
					"volumes": []map[string]any{
						{
							"name": CinderCephExtraMountsSecretName,
							"secret": map[string]any{
								"secretName": CinderCephExtraMountsSecretName,
							},
						},
					},
					"mounts": []map[string]any{
						{
							"name":      CinderCephExtraMountsSecretName,
							"mountPath": CinderCephExtraMountsPath,
							"readOnly":  true,
						},
					},
				},
			},
		},
	}
}

// Topology functions

// GetSampleTopologySpec - An opinionated Topology Spec sample used to
// test Service components. It returns both the user input representation
// in the form of map[string]string, and the Golang expected representation
// used in the test asserts.
func GetSampleTopologySpec(label string) (map[string]any, []corev1.TopologySpreadConstraint) {
	// Build the topology Spec
	topologySpec := map[string]any{
		"topologySpreadConstraints": []map[string]any{
			{
				"maxSkew":           1,
				"topologyKey":       corev1.LabelHostname,
				"whenUnsatisfiable": "ScheduleAnyway",
				"labelSelector": map[string]any{
					"matchLabels": map[string]any{
						"component": label,
					},
				},
			},
		},
	}
	// Build the topologyObj representation
	topologySpecObj := []corev1.TopologySpreadConstraint{
		{
			MaxSkew:           1,
			TopologyKey:       corev1.LabelHostname,
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"component": label,
				},
			},
		},
	}
	return topologySpec, topologySpecObj
}

// GetCinderSpecWithAC returns a Cinder spec with Application Credential configured
func GetCinderSpecWithAC(acSecretName string) map[string]interface{} {
	spec := GetDefaultCinderSpec()
	spec["secret"] = ACTestServicePasswordSecret
	spec["auth"] = map[string]interface{}{
		"applicationCredentialSecret": acSecretName,
	}
	return spec
}

// GetDefaultCinderAC returns a default KeystoneApplicationCredential spec for testing
func GetDefaultCinderAC(namespace string, acName string) *keystonev1.KeystoneApplicationCredential {
	return &keystonev1.KeystoneApplicationCredential{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      acName,
		},
		Spec: keystonev1.KeystoneApplicationCredentialSpec{
			UserName:         cinder.ServiceName,
			Secret:           ACTestServicePasswordSecret,
			PasswordSelector: ACTestPasswordSelector,
			Roles:            []string{"admin", "member"},
			AccessRules:      []keystonev1.ACRule{{Service: "identity", Method: "POST", Path: "/auth/tokens"}},
			ExpirationDays:   30,
			GracePeriodDays:  5,
		},
	}
}

// CreateACSecret creates an Application Credential secret for testing
func CreateACSecret(namespace string, secretName string) *corev1.Secret {
	return th.CreateSecret(
		types.NamespacedName{Namespace: namespace, Name: secretName},
		map[string][]byte{
			keystonev1.ACIDSecretKey:     []byte("test-ac-id"),
			keystonev1.ACSecretSecretKey: []byte("test-ac-secret"),
		},
	)
}

// GetKeystoneAC fetches a KeystoneApplicationCredential by name
func GetKeystoneAC(name types.NamespacedName) *keystonev1.KeystoneApplicationCredential {
	instance := &keystonev1.KeystoneApplicationCredential{}
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, name, instance)).Should(Succeed())
	}, timeout, interval).Should(Succeed())
	return instance
}

// GetProbeConfOverrides returns a set of parameters to override the default
// probes values
func GetProbeConfOverrides() map[string]any {
	return map[string]any{
		"path":                "/healthcheck",
		"initialDelaySeconds": int32(20),
		"timeoutSeconds":      int32(30),
		"periodSeconds":       int32(10),
	}
}
