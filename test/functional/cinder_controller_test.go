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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	cinderv1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	memcachedv1 "github.com/openstack-k8s-operators/infra-operator/apis/memcached/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	util "github.com/openstack-k8s-operators/lib-common/modules/common/util"
)

var _ = Describe("Cinder controller", func() {
	var memcachedSpec memcachedv1.MemcachedSpec

	BeforeEach(func() {
		memcachedSpec = memcachedv1.MemcachedSpec{
			Replicas: ptr.To(int32(3)),
		}
	})

	When("Cinder CR instance is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, GetDefaultCinderSpec()))
		})
		It("initializes the status fields", func() {
			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderName)
				g.Expect(cinder.Status.Conditions).To(HaveLen(16))

				g.Expect(cinder.Status.DatabaseHostname).To(Equal(""))
			}, timeout*2, interval).Should(Succeed())
		})
		It("is not Ready", func() {
			th.ExpectCondition(
				cinderTest.Instance,
				ConditionGetterFunc(CinderConditionGetter),
				condition.ReadyCondition,
				corev1.ConditionUnknown,
			)
		})
		It("should have the Spec fields initialized", func() {
			Cinder := GetCinder(cinderTest.Instance)
			Expect(Cinder.Spec.DatabaseInstance).Should(Equal("openstack"))
			Expect(Cinder.Spec.DatabaseUser).Should(Equal(cinderTest.CinderDataBaseUser))
			Expect(Cinder.Spec.MemcachedInstance).Should(Equal(cinderTest.MemcachedInstance))
			Expect(Cinder.Spec.RabbitMqClusterName).Should(Equal(cinderTest.RabbitmqClusterName))
			Expect(Cinder.Spec.ServiceUser).Should(Equal(cinderTest.CinderServiceUser))
		})
		It("should have the Status fields initialized", func() {
			Cinder := GetCinder(cinderTest.Instance)
			Expect(Cinder.Status.Hash).To(BeEmpty())
			Expect(Cinder.Status.DatabaseHostname).To(Equal(""))
			Expect(Cinder.Status.TransportURLSecret).To(Equal(""))
			Expect(Cinder.Status.CinderAPIReadyCount).To(Equal(int32(0)))
			Expect(Cinder.Status.CinderBackupReadyCount).To(Equal(int32(0)))
			Expect(Cinder.Status.CinderSchedulerReadyCount).To(Equal(int32(0)))
			Expect(Cinder.Status.CinderVolumesReadyCounts["volume1"]).To(Equal(int32(0)))
			Expect(Cinder.Status.CinderVolumesReadyCounts["volume2"]).To(Equal(int32(0)))
		})
		It("should have Unknown Conditions initialized", func() {
			for _, cond := range []condition.Type{
				condition.CronJobReadyCondition,
				condition.DBReadyCondition,
				condition.DBSyncReadyCondition,
				condition.InputReadyCondition,
				condition.MemcachedReadyCondition,
				cinderv1.CinderAPIReadyCondition,
				cinderv1.CinderBackupReadyCondition,
				cinderv1.CinderSchedulerReadyCondition,
				cinderv1.CinderVolumeReadyCondition,
			} {
				th.ExpectCondition(
					cinderTest.Cinder,
					ConditionGetterFunc(CinderConditionGetter),
					cond,
					corev1.ConditionUnknown,
				)
			}
		})
		It("should have a finalizer", func() {
			// the reconciler loop adds the finalizer so we have to wait for
			// it to run
			Eventually(func() []string {
				return GetCinder(cinderTest.Instance).Finalizers
			}, timeout, interval).Should(ContainElement("Cinder"))
		})
		It("creates service account, role and rolebinding", func() {
			th.ExpectCondition(
				cinderName,
				ConditionGetterFunc(CinderConditionGetter),
				condition.ServiceAccountReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				cinderName,
				ConditionGetterFunc(CinderConditionGetter),
				condition.RoleReadyCondition,
				corev1.ConditionTrue,
			)
			role := th.GetRole(cinderTest.CinderRole)
			Expect(role.Rules).To(HaveLen(2))
			Expect(role.Rules[0].Resources).To(Equal([]string{"securitycontextconstraints"}))
			Expect(role.Rules[1].Resources).To(Equal([]string{"pods"}))

			th.ExpectCondition(
				cinderName,
				ConditionGetterFunc(CinderConditionGetter),
				condition.RoleBindingReadyCondition,
				corev1.ConditionTrue,
			)

			sa := th.GetServiceAccount(cinderTest.CinderSA)

			binding := th.GetRoleBinding(cinderTest.CinderRoleBinding)
			Expect(binding.Subjects).To(HaveLen(1))
			Expect(binding.RoleRef.Name).To(Equal(role.Name))
			Expect(binding.Subjects[0].Name).To(Equal(sa.Name))
		})
	})
	When("Cinder DB is created", func() {
		BeforeEach(func() {
			DeferCleanup(k8sClient.Delete, ctx, CreateCinderMessageBusSecret(cinderTest.Instance.Namespace, cinderTest.RabbitmqSecretName))
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, GetDefaultCinderSpec()))
			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					namespace,
					GetCinder(cinderTest.Instance).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(cinderTest.CinderTransportURL)
			DeferCleanup(infra.DeleteMemcached, infra.CreateMemcached(namespace, cinderTest.MemcachedInstance, memcachedSpec))
			infra.SimulateMemcachedReady(cinderTest.CinderMemcached)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(namespace))
		})
		It("Should set DBReady Condition and set DatabaseHostname Status when DB is Created", func() {
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Instance)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
			Cinder := GetCinder(cinderTest.Instance)
			Expect(Cinder.Status.DatabaseHostname).To(Equal("hostname-for-openstack"))
			th.ExpectCondition(
				cinderName,
				ConditionGetterFunc(CinderConditionGetter),
				condition.DBReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				cinderName,
				ConditionGetterFunc(CinderConditionGetter),
				condition.DBSyncReadyCondition,
				corev1.ConditionFalse,
			)
		})
		It("Should fail if db-sync job fails when DB is Created", func() {
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Instance)
			th.SimulateJobFailure(cinderTest.CinderDBSync)
			th.ExpectCondition(
				cinderTest.Instance,
				ConditionGetterFunc(CinderConditionGetter),
				condition.DBReadyCondition,
				corev1.ConditionTrue,
			)
			th.ExpectCondition(
				cinderTest.Instance,
				ConditionGetterFunc(CinderConditionGetter),
				condition.DBSyncReadyCondition,
				corev1.ConditionFalse,
			)
		})
		It("Does not create CinderAPI", func() {
			CinderAPINotExists(cinderTest.Instance)
		})
		It("Does not create CinderScheduler", func() {
			CinderSchedulerNotExists(cinderTest.Instance)
		})
		It("Does not create CinderVolume", func() {
			CinderVolumeNotExists(cinderTest.Instance)
		})
	})
	When("Both TransportURL secret and osp-secret are available", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, GetDefaultCinderSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateCinderMessageBusSecret(cinderTest.Instance.Namespace, cinderTest.RabbitmqSecretName))
			DeferCleanup(mariadb.DeleteDBService, mariadb.CreateDBService(
				cinderTest.Instance.Namespace,
				GetCinder(cinderName).Spec.DatabaseInstance,
				corev1.ServiceSpec{
					Ports: []corev1.ServicePort{{Port: 3306}},
				},
			),
			)
			infra.SimulateTransportURLReady(cinderTest.CinderTransportURL)
			DeferCleanup(infra.DeleteMemcached, infra.CreateMemcached(namespace, "memcached", memcachedSpec))
			infra.SimulateMemcachedReady(cinderTest.CinderMemcached)
		})
		It("should create config-data and scripts ConfigMaps", func() {
			keystoneAPI := keystone.CreateKeystoneAPI(cinderTest.Instance.Namespace)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystoneAPI)
			Eventually(func() corev1.Secret {
				return th.GetSecret(cinderTest.CinderConfigSecret)
			}, timeout, interval).ShouldNot(BeNil())
			Eventually(func() corev1.Secret {
				return th.GetSecret(cinderTest.CinderConfigScripts)
			}, timeout, interval).ShouldNot(BeNil())
		})
	})
	When("Cinder CR is created without container images defined", func() {
		BeforeEach(func() {
			// CinderEmptySpec is used to provide a standard Cinder CR where no
			// field is customized
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, GetCinderEmptySpec()))
		})
		It("has the expected container image defaults", func() {
			cinderDefault := GetCinder(cinderTest.Instance)
			Expect(cinderDefault.Spec.CinderAPI.CinderServiceTemplate.ContainerImage).To(Equal(util.GetEnvVar("RELATED_IMAGE_CINDER_API_IMAGE_URL_DEFAULT", cinderv1.CinderAPIContainerImage)))
			Expect(cinderDefault.Spec.CinderScheduler.CinderServiceTemplate.ContainerImage).To(Equal(util.GetEnvVar("RELATED_IMAGE_CINDER_SCHEDULER_IMAGE_URL_DEFAULT", cinderv1.CinderSchedulerContainerImage)))
			for _, volume := range cinderDefault.Spec.CinderVolumes {
				Expect(volume.ContainerImage).To(Equal(util.GetEnvVar("RELATED_IMAGE_CINDER_VOLUME_IMAGE_URL_DEFAULT", cinderv1.CinderVolumeContainerImage)))
			}
		})
	})
	When("All the Resources are ready", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, GetDefaultCinderSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateCinderMessageBusSecret(cinderTest.Instance.Namespace, cinderTest.RabbitmqSecretName))
			DeferCleanup(th.DeleteInstance, CreateCinderAPI(cinderTest.Instance, GetDefaultCinderAPISpec()))
			DeferCleanup(th.DeleteInstance, CreateCinderScheduler(cinderTest.Instance, GetDefaultCinderSchedulerSpec()))
			DeferCleanup(th.DeleteInstance, CreateCinderVolume(cinderTest.Instance, GetDefaultCinderVolumeSpec()))
			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					cinderTest.Instance.Namespace,
					GetCinder(cinderName).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(cinderTest.CinderTransportURL)
			DeferCleanup(infra.DeleteMemcached, infra.CreateMemcached(namespace, cinderTest.MemcachedInstance, memcachedSpec))
			infra.SimulateMemcachedReady(cinderTest.CinderMemcached)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(cinderTest.Instance.Namespace))
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Instance)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)
		})
		It("Creates CinderAPI", func() {
			CinderAPIExists(cinderTest.Instance)
		})
		It("Creates CinderScheduler", func() {
			CinderSchedulerExists(cinderTest.Instance)
		})
		It("Creates CinderVolume", func() {
			CinderVolumeExists(cinderTest.Instance)
		})
		It("Assert Services are created", func() {
			th.AssertServiceExists(cinderTest.CinderServicePublic)
			th.AssertServiceExists(cinderTest.CinderServiceInternal)
		})
	})
	When("Cinder CR instance is deleted", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, GetDefaultCinderSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateCinderMessageBusSecret(cinderTest.Instance.Namespace, cinderTest.RabbitmqSecretName))
			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					cinderTest.Instance.Namespace,
					GetCinder(cinderTest.Instance).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(cinderTest.CinderTransportURL)
			DeferCleanup(infra.DeleteMemcached, infra.CreateMemcached(namespace, cinderTest.MemcachedInstance, memcachedSpec))
			infra.SimulateMemcachedReady(cinderTest.CinderMemcached)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(cinderTest.Instance.Namespace))
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Instance)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
		})
		It("removes the finalizers from the Cinder DB", func() {
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)

			mDB := mariadb.GetMariaDBDatabase(cinderTest.Instance)
			Expect(mDB.Finalizers).To(ContainElement("Cinder"))

			th.DeleteInstance(GetCinder(cinderTest.Instance))

			mDB = mariadb.GetMariaDBDatabase(cinderTest.Instance)
			Expect(mDB.Finalizers).NotTo(ContainElement("Cinder"))
		})
	})
	When("Cinder CR instance is built with NAD", func() {
		BeforeEach(func() {
			nad := th.CreateNetworkAttachmentDefinition(cinderTest.InternalAPINAD)
			DeferCleanup(th.DeleteInstance, nad)
			serviceOverride := map[string]interface{}{}
			serviceOverride["internal"] = map[string]interface{}{
				"metadata": map[string]map[string]string{
					"annotations": {
						"metallb.universe.tf/address-pool":     "osp-internalapi",
						"metallb.universe.tf/allow-volumed-ip": "osp-internalapi",
						"metallb.universe.tf/loadBalancerIPs":  "internal-lb-ip-1,internal-lb-ip-2",
					},
					"labels": {
						"internal": "true",
						"service":  "nova",
					},
				},
				"spec": map[string]interface{}{
					"type": "LoadBalancer",
				},
			}

			rawSpec := map[string]interface{}{
				"secret":              SecretName,
				"databaseInstance":    "openstack",
				"rabbitMqClusterName": "rabbitmq",
				"cinderAPI": map[string]interface{}{
					"containerImage":     cinderv1.CinderAPIContainerImage,
					"networkAttachments": []string{"internalapi"},
					"override": map[string]interface{}{
						"service": serviceOverride,
					},
				},
				"cinderScheduler": map[string]interface{}{
					"containerImage":     cinderv1.CinderSchedulerContainerImage,
					"networkAttachments": []string{"internalapi"},
				},
				"cinderVolumes": map[string]interface{}{
					"volume1": map[string]interface{}{
						"containerImage":     cinderv1.CinderVolumeContainerImage,
						"networkAttachments": []string{"internalapi"},
					},
				},
			}
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, rawSpec))
			DeferCleanup(k8sClient.Delete, ctx, CreateCinderMessageBusSecret(cinderTest.Instance.Namespace, cinderTest.RabbitmqSecretName))
			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					cinderTest.Instance.Namespace,
					GetCinder(cinderTest.Instance).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(cinderTest.CinderTransportURL)
			DeferCleanup(infra.DeleteMemcached, infra.CreateMemcached(namespace, cinderTest.MemcachedInstance, memcachedSpec))
			infra.SimulateMemcachedReady(cinderTest.CinderMemcached)
			keystoneAPIName := keystone.CreateKeystoneAPI(cinderTest.Instance.Namespace)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystoneAPIName)
			keystoneAPI := keystone.GetKeystoneAPI(keystoneAPIName)
			keystoneAPI.Status.APIEndpoints["internal"] = "http://keystone-internal-openstack.testing"
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Status().Update(ctx, keystoneAPI.DeepCopy())).Should(Succeed())
			}, timeout, interval).Should(Succeed())
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Instance)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
		})
		It("Check the resulting endpoints of the generated sub-CRs", func() {
			th.SimulateStatefulSetReplicaReadyWithPods(
				cinderTest.CinderAPI,
				map[string][]string{cinderName.Namespace + "/internalapi": {"10.0.0.1"}},
			)
			th.SimulateStatefulSetReplicaReadyWithPods(
				cinderTest.CinderScheduler,
				map[string][]string{cinderName.Namespace + "/internalapi": {"10.0.0.1"}},
			)
			th.SimulateStatefulSetReplicaReadyWithPods(
				cinderTest.CinderVolumes[0],
				map[string][]string{cinderName.Namespace + "/internalapi": {"10.0.0.1"}},
			)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)
			// Retrieve the generated resources
			cinder := GetCinder(cinderTest.Instance)
			api := GetCinderAPI(cinderTest.CinderAPI)
			sched := GetCinderScheduler(cinderTest.CinderScheduler)
			volume := GetCinderVolume(cinderTest.CinderVolumes[0])
			// Check CinderAPI NADs
			Expect(api.Spec.NetworkAttachments).To(Equal(cinder.Spec.CinderAPI.CinderServiceTemplate.NetworkAttachments))
			// Check CinderScheduler NADs
			Expect(sched.Spec.NetworkAttachments).To(Equal(cinder.Spec.CinderScheduler.CinderServiceTemplate.NetworkAttachments))
			// Check CinderVolume exists
			CinderVolumeExists(cinderTest.CinderVolumes[0])
			// Check CinderVolume NADs
			Expect(volume.Spec.NetworkAttachments).To(Equal(volume.Spec.CinderServiceTemplate.NetworkAttachments))

			// As the internal endpoint has service override configured it
			// gets a LoadBalancer Service with MetalLB annotations
			service := th.GetService(cinderTest.CinderServiceInternal)
			Expect(service.Spec.Type).To(Equal(corev1.ServiceTypeLoadBalancer))
			Expect(service.Annotations).To(
				HaveKeyWithValue("dnsmasq.network.openstack.org/hostname", "cinder-internal."+cinder.Namespace+".svc"))
			Expect(service.Annotations).To(
				HaveKeyWithValue("metallb.universe.tf/address-pool", "osp-internalapi"))
			Expect(service.Annotations).To(
				HaveKeyWithValue("metallb.universe.tf/allow-volumed-ip", "osp-internalapi"))
			Expect(service.Annotations).To(
				HaveKeyWithValue("metallb.universe.tf/loadBalancerIPs", "internal-lb-ip-1,internal-lb-ip-2"))

			// check keystone endpoints
			keystoneEndpoint := keystone.GetKeystoneEndpoint(cinderTest.CinderKeystoneEndpoint)
			endpoints := keystoneEndpoint.Spec.Endpoints
			Expect(endpoints).To(HaveKeyWithValue("public", "http://cinder-public."+cinder.Namespace+".svc:8776/v3"))
			Expect(endpoints).To(HaveKeyWithValue("internal", "http://cinder-internal."+cinder.Namespace+".svc:8776/v3"))
		})
	})
	When("A Cinder with TLS is created", func() {
		BeforeEach(func() {
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, GetTLSCinderSpec()))
			DeferCleanup(k8sClient.Delete, ctx, CreateCinderMessageBusSecret(cinderTest.Instance.Namespace, cinderTest.RabbitmqSecretName))
			DeferCleanup(th.DeleteInstance, CreateCinderAPI(cinderTest.Instance, GetDefaultCinderAPISpec()))
			DeferCleanup(th.DeleteInstance, CreateCinderScheduler(cinderTest.Instance, GetDefaultCinderSchedulerSpec()))
			DeferCleanup(th.DeleteInstance, CreateCinderVolume(cinderTest.Instance, GetDefaultCinderVolumeSpec()))
			DeferCleanup(
				mariadb.DeleteDBService,
				mariadb.CreateDBService(
					cinderTest.Instance.Namespace,
					GetCinder(cinderName).Spec.DatabaseInstance,
					corev1.ServiceSpec{
						Ports: []corev1.ServicePort{{Port: 3306}},
					},
				),
			)
			infra.SimulateTransportURLReady(cinderTest.CinderTransportURL)
			DeferCleanup(infra.DeleteMemcached, infra.CreateMemcached(namespace, cinderTest.MemcachedInstance, memcachedSpec))
			infra.SimulateMemcachedReady(cinderTest.CinderMemcached)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystone.CreateKeystoneAPI(cinderTest.Instance.Namespace))
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Instance)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
		})

		It("reports that the CA secret is missing", func() {
			th.ExpectConditionWithDetails(
				cinderTest.CinderAPI,
				ConditionGetterFunc(CinderAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionFalse,
				condition.ErrorReason,
				fmt.Sprintf("TLSInput error occured in TLS sources Secret %s/combined-ca-bundle not found", namespace),
			)

			th.ExpectConditionWithDetails(
				cinderTest.CinderScheduler,
				ConditionGetterFunc(CinderSchedulerConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionFalse,
				condition.ErrorReason,
				fmt.Sprintf("TLSInput error occured in TLS sources Secret %s/combined-ca-bundle not found", namespace),
			)
		})

		It("reports that the internal cert secret is missing", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			th.ExpectConditionWithDetails(
				cinderTest.CinderAPI,
				ConditionGetterFunc(CinderAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionFalse,
				condition.ErrorReason,
				fmt.Sprintf("TLSInput error occured in TLS sources Secret %s/internal-tls-certs not found", namespace),
			)
		})

		It("reports that the public cert secret is missing", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.InternalCertSecret))
			th.ExpectConditionWithDetails(
				cinderTest.CinderAPI,
				ConditionGetterFunc(CinderAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionFalse,
				condition.ErrorReason,
				fmt.Sprintf("TLSInput error occured in TLS sources Secret %s/public-tls-certs not found", namespace),
			)
		})

		It("Creates CinderAPI", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.PublicCertSecret))
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)

			CinderAPIExists(cinderTest.Instance)

			d := th.GetStatefulSet(cinderTest.CinderAPI)
			// Check the resulting deployment fields
			Expect(int(*d.Spec.Replicas)).To(Equal(1))
			Expect(d.Spec.Template.Spec.Volumes).To(HaveLen(9))
			Expect(d.Spec.Template.Spec.Containers).To(HaveLen(2))

			// cert deployment volumes
			th.AssertVolumeExists(cinderTest.CABundleSecret.Name, d.Spec.Template.Spec.Volumes)
			th.AssertVolumeExists(cinderTest.InternalCertSecret.Name, d.Spec.Template.Spec.Volumes)
			th.AssertVolumeExists(cinderTest.PublicCertSecret.Name, d.Spec.Template.Spec.Volumes)

			// cert volumeMounts
			container := d.Spec.Template.Spec.Containers[1]
			th.AssertVolumeMountExists(cinderTest.InternalCertSecret.Name, "tls.key", container.VolumeMounts)
			th.AssertVolumeMountExists(cinderTest.InternalCertSecret.Name, "tls.crt", container.VolumeMounts)
			th.AssertVolumeMountExists(cinderTest.PublicCertSecret.Name, "tls.key", container.VolumeMounts)
			th.AssertVolumeMountExists(cinderTest.PublicCertSecret.Name, "tls.crt", container.VolumeMounts)
			th.AssertVolumeMountExists(cinderTest.CABundleSecret.Name, "tls-ca-bundle.pem", container.VolumeMounts)

			Expect(container.ReadinessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTPS))
			Expect(container.LivenessProbe.HTTPGet.Scheme).To(Equal(corev1.URISchemeHTTPS))
		})
		It("Creates CinderScheduler", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.PublicCertSecret))
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)

			CinderSchedulerExists(cinderTest.Instance)

			ss := th.GetStatefulSet(cinderTest.CinderScheduler)
			// Check the resulting deployment fields
			Expect(int(*ss.Spec.Replicas)).To(Equal(1))
			Expect(ss.Spec.Template.Spec.Volumes).To(HaveLen(6))
			Expect(ss.Spec.Template.Spec.Containers).To(HaveLen(2))

			// cert deployment volumes
			th.AssertVolumeExists(cinderTest.CABundleSecret.Name, ss.Spec.Template.Spec.Volumes)

			// cert volumeMounts
			container := ss.Spec.Template.Spec.Containers[1]
			th.AssertVolumeMountExists(cinderTest.CABundleSecret.Name, "tls-ca-bundle.pem", container.VolumeMounts)
		})
		It("Creates CinderVolume", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.PublicCertSecret))
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)

			CinderVolumeExists(cinderTest.Instance)
			// follow up to check CA cert moun on c-v when fully integrated in functional tests
		})
		It("Assert Services are created", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.PublicCertSecret))
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)

			th.AssertServiceExists(cinderTest.CinderServicePublic)
			th.AssertServiceExists(cinderTest.CinderServiceInternal)

			// check keystone endpoints
			keystoneEndpoint := keystone.GetKeystoneEndpoint(cinderTest.CinderKeystoneEndpoint)
			endpoints := keystoneEndpoint.Spec.Endpoints
			Expect(endpoints).To(HaveKeyWithValue("public", "https://cinder-public."+namespace+".svc:8776/v3"))
			Expect(endpoints).To(HaveKeyWithValue("internal", "https://cinder-internal."+namespace+".svc:8776/v3"))
		})
	})
})
