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
	"errors"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2" //revive:disable:dot-imports
	. "github.com/onsi/gomega"    //revive:disable:dot-imports
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	//revive:disable-next-line:dot-imports
	. "github.com/openstack-k8s-operators/lib-common/modules/common/test/helpers"

	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	cinderv1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	memcachedv1 "github.com/openstack-k8s-operators/infra-operator/apis/memcached/v1beta1"
	condition "github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	util "github.com/openstack-k8s-operators/lib-common/modules/common/util"
	mariadb_test "github.com/openstack-k8s-operators/mariadb-operator/api/test/helpers"
)

var _ = Describe("Cinder controller", func() {
	var memcachedSpec memcachedv1.MemcachedSpec

	BeforeEach(func() {
		memcachedSpec = memcachedv1.MemcachedSpec{
			MemcachedSpecCore: memcachedv1.MemcachedSpecCore{
				Replicas: ptr.To(int32(3)),
			},
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
				corev1.ConditionFalse,
			)
		})
		It("should have the Spec fields initialized", func() {
			Cinder := GetCinder(cinderTest.Instance)
			Expect(Cinder.Spec.DatabaseInstance).Should(Equal("openstack"))
			Expect(Cinder.Spec.DatabaseAccount).Should(Equal(cinderTest.CinderDataBaseAccount))
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
			}, timeout, interval).Should(ContainElement("openstack.org/cinder"))
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
			Cinder := GetCinder(cinderTest.Instance)
			Expect(Cinder.Status.DatabaseHostname).To(Equal(fmt.Sprintf("hostname-for-openstack.%s.svc", namespace)))
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
		})
		It("should create config-data and scripts ConfigMaps", func() {
			keystoneAPI := keystone.CreateKeystoneAPI(cinderTest.Instance.Namespace)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystoneAPI)
			cf := th.GetSecret(cinderTest.CinderConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			conf := cf.Data[cinder.MyCnfFileName]
			Expect(conf).To(
				ContainSubstring("[client]\nssl=0"))
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
			Expect(cinderDefault.Spec.CinderAPI.ContainerImage).To(Equal(util.GetEnvVar("RELATED_IMAGE_CINDER_API_IMAGE_URL_DEFAULT", cinderv1.CinderAPIContainerImage)))
			Expect(cinderDefault.Spec.CinderScheduler.ContainerImage).To(Equal(util.GetEnvVar("RELATED_IMAGE_CINDER_SCHEDULER_IMAGE_URL_DEFAULT", cinderv1.CinderSchedulerContainerImage)))
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
		})
		It("removes the finalizers from the Cinder DB", func() {
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)

			mDB := mariadb.GetMariaDBDatabase(cinderTest.Database)
			Expect(mDB.Finalizers).To(ContainElement("openstack.org/cinder"))

			th.DeleteInstance(GetCinder(cinderTest.Instance))

			mDB = mariadb.GetMariaDBDatabase(cinderTest.Database)
			Expect(mDB.Finalizers).NotTo(ContainElement("openstack.org/cinder"))
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
			th.SimulateLoadBalancerServiceIP(cinderTest.CinderServiceInternal)
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBTLSDatabaseCompleted(cinderTest.Database)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
		})

		It("reports that the CA secret is missing", func() {
			th.ExpectCondition(
				cinderTest.CinderAPI,
				ConditionGetterFunc(CinderAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionFalse,
			)

			th.ExpectCondition(
				cinderTest.CinderScheduler,
				ConditionGetterFunc(CinderSchedulerConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("reports that the internal cert secret is missing", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			th.ExpectCondition(
				cinderTest.CinderAPI,
				ConditionGetterFunc(CinderAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("reports that the public cert secret is missing", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.InternalCertSecret))
			th.ExpectCondition(
				cinderTest.CinderAPI,
				ConditionGetterFunc(CinderAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionFalse,
			)
		})

		It("should create config-data and scripts ConfigMaps", func() {
			keystoneAPI := keystone.CreateKeystoneAPI(cinderTest.Instance.Namespace)
			DeferCleanup(keystone.DeleteKeystoneAPI, keystoneAPI)
			cf := th.GetSecret(cinderTest.CinderConfigSecret)
			Expect(cf).ShouldNot(BeNil())
			conf := cf.Data[cinder.MyCnfFileName]
			Expect(conf).To(
				ContainSubstring("[client]\nssl-ca=/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem\nssl=1"))
			Eventually(func() corev1.Secret {
				return th.GetSecret(cinderTest.CinderConfigScripts)
			}, timeout, interval).ShouldNot(BeNil())
		})

		It("Creates CinderAPI", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.PublicCertSecret))
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)

			CinderAPIExists(cinderTest.Instance)

			th.ExpectCondition(
				cinderTest.CinderAPI,
				ConditionGetterFunc(CinderAPIConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionTrue,
			)

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

			th.ExpectCondition(
				cinderTest.CinderScheduler,
				ConditionGetterFunc(CinderSchedulerConditionGetter),
				condition.TLSInputReadyCondition,
				corev1.ConditionTrue,
			)

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

		It("reconfigures the cinder pods when CA changes", func() {
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCABundleSecret(cinderTest.CABundleSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.InternalCertSecret))
			DeferCleanup(k8sClient.Delete, ctx, th.CreateCertSecret(cinderTest.PublicCertSecret))
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)

			CinderAPIExists(cinderTest.Instance)
			CinderSchedulerExists(cinderTest.Instance)
			CinderVolumeExists(cinderTest.Instance)

			// Grab the current config hash
			apiOriginalHash := GetEnvVarValue(
				th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.Containers[0].Env, "CONFIG_HASH", "")
			Expect(apiOriginalHash).NotTo(BeEmpty())
			schedulerOriginalHash := GetEnvVarValue(
				th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.Containers[0].Env, "CONFIG_HASH", "")
			Expect(schedulerOriginalHash).NotTo(BeEmpty())

			// Change the content of the CA secret
			th.UpdateSecret(cinderTest.CABundleSecret, "tls-ca-bundle.pem", []byte("DifferentCAData"))

			// Assert that the deployment is updated
			Eventually(func(g Gomega) {
				newHash := GetEnvVarValue(
					th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.Containers[0].Env, "CONFIG_HASH", "")
				g.Expect(newHash).NotTo(BeEmpty())
				g.Expect(newHash).NotTo(Equal(apiOriginalHash))
			}, timeout, interval).Should(Succeed())
			Eventually(func(g Gomega) {
				newHash := GetEnvVarValue(
					th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.Containers[0].Env, "CONFIG_HASH", "")
				g.Expect(newHash).NotTo(BeEmpty())
				g.Expect(newHash).NotTo(Equal(schedulerOriginalHash))
			}, timeout, interval).Should(Succeed())
		})
	})
	When("Cinder is created with topologyRef", func() {
		BeforeEach(func() {
			// Build the topology Spec
			topologySpec := GetSampleTopologySpec()
			// Create Test Topologies
			for _, t := range cinderTest.CinderTopologies {
				CreateTopology(t, topologySpec)
			}
			spec := GetDefaultCinderSpec()
			spec["topologyRef"] = map[string]interface{}{
				"name": cinderTest.CinderTopologies[0].Name,
			}
			// Override topologyRef for cinderVolume subCR
			spec["cinderVolumes"] = map[string]interface{}{
				"volume1": map[string]interface{}{},
			}

			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, spec))
			DeferCleanup(k8sClient.Delete, ctx, CreateCinderMessageBusSecret(cinderTest.Instance.Namespace, cinderTest.RabbitmqSecretName))
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)
			th.SimulateStatefulSetReplicaReady(cinderTest.CinderAPI)
			th.SimulateStatefulSetReplicaReady(cinderTest.CinderScheduler)
			th.SimulateStatefulSetReplicaReady(cinderTest.CinderVolumes[0])
		})
		It("sets topology in CR status", func() {
			Eventually(func(g Gomega) {
				cinderAPI := GetCinderAPI(cinderTest.CinderAPI)
				g.Expect(cinderAPI.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[0].Name))
				cinderScheduler := GetCinderScheduler(cinderTest.CinderScheduler)
				g.Expect(cinderScheduler.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[0].Name))
				cinderVolume := GetCinderVolume(cinderTest.CinderVolumes[0])
				g.Expect(cinderVolume.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[0].Name))
			}, timeout, interval).Should(Succeed())
		})
		It("sets Topology in resource specs", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.TopologySpreadConstraints).ToNot(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.Affinity).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.TopologySpreadConstraints).ToNot(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.Affinity).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.TopologySpreadConstraints).ToNot(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.Affinity).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})
		It("updates topology when the reference changes", func() {
			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderTest.Instance)
				cinder.Spec.TopologyRef.Name = cinderTest.CinderTopologies[1].Name
				g.Expect(k8sClient.Update(ctx, cinder)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				cinderAPI := GetCinderAPI(cinderTest.CinderAPI)
				g.Expect(cinderAPI.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[1].Name))
				cinderScheduler := GetCinderScheduler(cinderTest.CinderScheduler)
				g.Expect(cinderScheduler.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[1].Name))
				cinderVolume := GetCinderVolume(cinderTest.CinderVolumes[0])
				g.Expect(cinderVolume.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[1].Name))
			}, timeout, interval).Should(Succeed())
		})
		It("overrides topology when the reference changes", func() {
			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderTest.Instance)
				//Patch CinderAPI Spec
				newAPI := GetCinderAPISpec(cinderTest.CinderAPI)
				newAPI.TopologyRef.Name = cinderTest.CinderTopologies[1].Name
				cinder.Spec.CinderAPI = newAPI
				//Patch CinderScheduler Spec
				newSch := GetCinderSchedulerSpec(cinderTest.CinderScheduler)
				newSch.TopologyRef.Name = cinderTest.CinderTopologies[2].Name
				cinder.Spec.CinderScheduler = newSch
				// Patch CinderVolume (volume1) Spec
				newVol := GetCinderVolumeSpec(cinderTest.CinderVolumes[0])
				newVol.TopologyRef.Name = cinderTest.CinderTopologies[3].Name
				cinder.Spec.CinderVolumes["volume1"] = newVol
				g.Expect(k8sClient.Update(ctx, cinder)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				cinderAPI := GetCinderAPI(cinderTest.CinderAPI)
				g.Expect(cinderAPI.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[1].Name))
				cinderScheduler := GetCinderScheduler(cinderTest.CinderScheduler)
				g.Expect(cinderScheduler.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[2].Name))
				cinderVolume := GetCinderVolume(cinderTest.CinderVolumes[0])
				g.Expect(cinderVolume.Status.LastAppliedTopology.Name).To(Equal(cinderTest.CinderTopologies[3].Name))
			}, timeout, interval).Should(Succeed())
		})
		It("removes topologyRef from the spec", func() {
			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderTest.Instance)
				// Remove the TopologyRef from the existing Cinder .Spec
				cinder.Spec.TopologyRef = nil
				g.Expect(k8sClient.Update(ctx, cinder)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				cinderAPI := GetCinderAPI(cinderTest.CinderAPI)
				g.Expect(cinderAPI.Status.LastAppliedTopology).Should(BeNil())
				cinderScheduler := GetCinderScheduler(cinderTest.CinderScheduler)
				g.Expect(cinderScheduler.Status.LastAppliedTopology).Should(BeNil())
				cinderVolume := GetCinderVolume(cinderTest.CinderVolumes[0])
				g.Expect(cinderVolume.Status.LastAppliedTopology).Should(BeNil())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.TopologySpreadConstraints).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.Affinity).ToNot(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.TopologySpreadConstraints).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.Affinity).ToNot(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.TopologySpreadConstraints).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.Affinity).ToNot(BeNil())
			}, timeout, interval).Should(Succeed())
		})
	})

	When("A Cinder is created with nodeSelector", func() {
		BeforeEach(func() {
			spec := GetDefaultCinderSpec()
			spec["nodeSelector"] = map[string]interface{}{
				"foo": "bar",
			}
			spec["cinderVolumes"] = map[string]interface{}{
				"volume1": map[string]interface{}{
					"containerImage": cinderv1.CinderVolumeContainerImage,
				},
			}
			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, spec))
			DeferCleanup(k8sClient.Delete, ctx, CreateCinderMessageBusSecret(cinderTest.Instance.Namespace, cinderTest.RabbitmqSecretName))
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
			keystone.SimulateKeystoneServiceReady(cinderTest.CinderKeystoneService)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)
			th.SimulateStatefulSetReplicaReady(cinderTest.CinderAPI)
			th.SimulateStatefulSetReplicaReady(cinderTest.CinderScheduler)
			th.SimulateStatefulSetReplicaReady(cinderTest.CinderVolumes[0])
		})

		It("sets nodeSelector in resource specs", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())
		})

		It("updates nodeSelector in resource specs when changed", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderName)
				newNodeSelector := map[string]string{
					"foo2": "bar2",
				}
				cinder.Spec.NodeSelector = &newNodeSelector
				g.Expect(k8sClient.Update(ctx, cinder)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(cinderTest.CinderDBSync)
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo2": "bar2"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo2": "bar2"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo2": "bar2"}))
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo2": "bar2"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo2": "bar2"}))
			}, timeout, interval).Should(Succeed())
		})

		It("removes nodeSelector from resource specs when cleared", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderName)
				emptyNodeSelector := map[string]string{}
				cinder.Spec.NodeSelector = &emptyNodeSelector
				g.Expect(k8sClient.Update(ctx, cinder)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(cinderTest.CinderDBSync)
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})

		It("removes nodeSelector from resource specs when nilled", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderName)
				cinder.Spec.NodeSelector = nil
				g.Expect(k8sClient.Update(ctx, cinder)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(cinderTest.CinderDBSync)
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(BeNil())
			}, timeout, interval).Should(Succeed())
		})

		It("allows nodeSelector service override", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderName)
				apiNodeSelector := map[string]string{
					"foo": "api",
				}
				cinder.Spec.CinderAPI.NodeSelector = &apiNodeSelector
				schedulerNodeSelector := map[string]string{
					"foo": "scheduler",
				}
				cinder.Spec.CinderScheduler.NodeSelector = &schedulerNodeSelector
				volumeNodeSelector := map[string]string{
					"foo": "volume",
				}
				volume := cinder.Spec.CinderVolumes["volume1"]
				volume.NodeSelector = &volumeNodeSelector
				cinder.Spec.CinderVolumes["volume1"] = volume
				g.Expect(k8sClient.Update(ctx, cinder)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(cinderTest.CinderDBSync)
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "api"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "scheduler"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "volume"}))
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())
		})

		It("allows nodeSelector service override to empty", func() {
			Eventually(func(g Gomega) {
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderName)
				apiNodeSelector := map[string]string{}
				cinder.Spec.CinderAPI.NodeSelector = &apiNodeSelector
				schedulerNodeSelector := map[string]string{}
				cinder.Spec.CinderScheduler.NodeSelector = &schedulerNodeSelector
				volumeNodeSelector := map[string]string{}
				volume := cinder.Spec.CinderVolumes["volume1"]
				volume.NodeSelector = &volumeNodeSelector
				cinder.Spec.CinderVolumes["volume1"] = volume
				g.Expect(k8sClient.Update(ctx, cinder)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				th.SimulateJobSuccess(cinderTest.CinderDBSync)
				g.Expect(th.GetStatefulSet(cinderTest.CinderAPI).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderScheduler).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetStatefulSet(cinderTest.CinderVolumes[0]).Spec.Template.Spec.NodeSelector).To(BeNil())
				g.Expect(th.GetJob(cinderTest.CinderDBSync).Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
				g.Expect(GetCronJob(cinderTest.CinderDBPurge).Spec.JobTemplate.Spec.Template.Spec.NodeSelector).To(Equal(map[string]string{"foo": "bar"}))
			}, timeout, interval).Should(Succeed())
		})

	})

	When("Cinder CR instance is built with ExtraMounts", func() {
		BeforeEach(func() {
			rawSpec := map[string]interface{}{
				"secret":              SecretName,
				"databaseInstance":    "openstack",
				"rabbitMqClusterName": "rabbitmq",
				"extraMounts":         GetExtraMounts(),
				"cinderAPI": map[string]interface{}{
					"containerImage": cinderv1.CinderAPIContainerImage,
				},
				"cinderScheduler": map[string]interface{}{
					"containerImage": cinderv1.CinderSchedulerContainerImage,
				},
				"cinderVolumes": map[string]interface{}{
					"volume1": map[string]interface{}{
						"containerImage": cinderv1.CinderVolumeContainerImage,
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
			mariadb.SimulateMariaDBAccountCompleted(cinderTest.Database)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)
			keystone.SimulateKeystoneEndpointReady(cinderTest.CinderKeystoneEndpoint)
		})

		It("Check the extraMounts of the resulting StatefulSets", func() {
			th.SimulateStatefulSetReplicaReady(cinderTest.CinderAPI)
			th.SimulateStatefulSetReplicaReady(cinderTest.CinderScheduler)
			// Retrieve the generated resources
			volume := cinderTest.CinderVolumes[0]
			th.SimulateStatefulSetReplicaReady(volume)
			ss := th.GetStatefulSet(volume)
			// Check the resulting deployment replicas
			Expect(int(*ss.Spec.Replicas)).To(Equal(1))
			// Assert Volume exists in the StatefulSet
			th.AssertVolumeExists(CinderCephExtraMountsSecretName, ss.Spec.Template.Spec.Volumes)
			// Get the cinder-volume container
			Expect(ss.Spec.Template.Spec.Containers).To(HaveLen(2))
			container := ss.Spec.Template.Spec.Containers[1]
			// Inspect VolumeMounts and make sure we have the Ceph MountPath
			// provided through extraMounts
			th.AssertVolumeMountExists(CinderCephExtraMountsSecretName, "", container.VolumeMounts)
		})
	})
	// Run MariaDBAccount suite tests.  these are pre-packaged ginkgo tests
	// that exercise standard account create / update patterns that should be
	// common to all controllers that ensure MariaDBAccount CRs.
	mariadbSuite := &mariadb_test.MariaDBTestHarness{
		PopulateHarness: func(harness *mariadb_test.MariaDBTestHarness) {
			harness.Setup(
				"Cinder",
				cinderTest.Instance.Namespace,
				cinder.DatabaseName,
				"openstack.org/cinder",
				mariadb,
				timeout,
				interval,
			)
		},
		// Generate a fully running Cinder service given an accountName
		// needs to make it all the way to the end where the mariadb finalizers
		// are removed from unused accounts since that's part of what we are testing
		SetupCR: func(accountName types.NamespacedName) {
			spec := GetTLSCinderSpec()
			spec["databaseAccount"] = accountName.Name

			DeferCleanup(th.DeleteInstance, CreateCinder(cinderTest.Instance, spec))
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
			mariadb.SimulateMariaDBAccountCompleted(accountName)
			mariadb.SimulateMariaDBDatabaseCompleted(cinderTest.Database)
			th.SimulateJobSuccess(cinderTest.CinderDBSync)

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

		},
		// Change the account name in the service to a new name
		UpdateAccount: func(newAccountName types.NamespacedName) {

			Eventually(func(g Gomega) {
				cinder := GetCinder(cinderName)
				cinder.Spec.DatabaseAccount = newAccountName.Name
				g.Expect(th.K8sClient.Update(ctx, cinder)).Should(Succeed())
			}, timeout, interval).Should(Succeed())

		},
		// delete the instance to exercise finalizer removal
		DeleteCR: func() {
			th.DeleteInstance(GetCinder(cinderName))
		},
	}

	mariadbSuite.RunBasicSuite()

})

var _ = Describe("Cinder Webhook", func() {

	BeforeEach(func() {
		err := os.Setenv("OPERATOR_TEMPLATES", "../../templates")
		Expect(err).NotTo(HaveOccurred())
	})

	It("rejects with wrong CinderAPI service override endpoint type", func() {
		spec := GetDefaultCinderSpec()
		apiSpec := GetDefaultCinderAPISpec()
		apiSpec["override"] = map[string]interface{}{
			"service": map[string]interface{}{
				"internal": map[string]interface{}{},
				"wrooong":  map[string]interface{}{},
			},
		}
		spec["cinderAPI"] = apiSpec

		raw := map[string]interface{}{
			"apiVersion": "cinder.openstack.org/v1beta1",
			"kind":       "Cinder",
			"metadata": map[string]interface{}{
				"name":      cinderTest.Instance.Name,
				"namespace": cinderTest.Instance.Namespace,
			},
			"spec": spec,
		}

		unstructuredObj := &unstructured.Unstructured{Object: raw}
		_, err := controllerutil.CreateOrPatch(
			th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(
			ContainSubstring(
				"invalid: spec.cinderAPI.override.service[wrooong]: " +
					"Invalid value: \"wrooong\": invalid endpoint type: wrooong"),
		)
	})

	It("webhooks reject the request - cinderVolume key too long", func() {
		spec := GetDefaultCinderSpec()
		raw := map[string]interface{}{
			"apiVersion": "cinder.openstack.org/v1beta1",
			"kind":       "Cinder",
			"metadata": map[string]interface{}{
				"name":      cinderTest.Instance.Name,
				"namespace": cinderTest.Instance.Namespace,
			},
			"spec": spec,
		}

		volumeList := map[string]interface{}{
			"foo-1234567890-1234567890-1234567890-1234567890-1234567890": GetDefaultCinderVolumeSpec(),
		}
		spec["cinderVolumes"] = volumeList

		unstructuredObj := &unstructured.Unstructured{Object: raw}
		_, err := controllerutil.CreateOrPatch(
			ctx, k8sClient, unstructuredObj, func() error { return nil })

		Expect(err).Should(HaveOccurred())
		var statusError *k8s_errors.StatusError
		Expect(errors.As(err, &statusError)).To(BeTrue())
		Expect(statusError.ErrStatus.Message).To(
			ContainSubstring(
				"Invalid value: \"foo-1234567890-1234567890-1234567890-1234567890-1234567890\": must be no more than 32 characters"),
		)
	})

	It("webhooks reject the request - cinderVolume wrong key/name", func() {
		spec := GetDefaultCinderSpec()
		raw := map[string]interface{}{
			"apiVersion": "cinder.openstack.org/v1beta1",
			"kind":       "Cinder",
			"metadata": map[string]interface{}{
				"name":      cinderTest.Instance.Name,
				"namespace": cinderTest.Instance.Namespace,
			},
			"spec": spec,
		}

		volumeList := map[string]interface{}{
			"foo_bar": GetDefaultCinderVolumeSpec(),
		}
		spec["cinderVolumes"] = volumeList

		unstructuredObj := &unstructured.Unstructured{Object: raw}
		_, err := controllerutil.CreateOrPatch(
			ctx, k8sClient, unstructuredObj, func() error { return nil })

		Expect(err).Should(HaveOccurred())
		var statusError *k8s_errors.StatusError
		Expect(errors.As(err, &statusError)).To(BeTrue())
		Expect(statusError.ErrStatus.Message).To(
			ContainSubstring(
				"Invalid value: \"foo_bar\": a lowercase RFC 1123 label must consist of lower case alphanumeric characters"),
		)
	})

	It("rejects a wrong TopologyRef on a different namespace", func() {
		spec := GetDefaultCinderSpec()
		// Reference a top-level topology
		spec["topologyRef"] = map[string]interface{}{
			"name":      cinderTest.CinderTopologies[0].Name,
			"namespace": "foo",
		}
		raw := map[string]interface{}{
			"apiVersion": "cinder.openstack.org/v1beta1",
			"kind":       "Cinder",
			"metadata": map[string]interface{}{
				"name":      cinderTest.Instance.Name,
				"namespace": cinderTest.Instance.Namespace,
			},
			"spec": spec,
		}

		unstructuredObj := &unstructured.Unstructured{Object: raw}
		_, err := controllerutil.CreateOrPatch(
			th.Ctx, th.K8sClient, unstructuredObj, func() error { return nil })
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(
			ContainSubstring(
				"Invalid value: \"namespace\": Customizing namespace field is not supported"),
		)
	})
})
