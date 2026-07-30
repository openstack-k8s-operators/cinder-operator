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

package cindervolume

import (
	cinderv1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	cinder "github.com/openstack-k8s-operators/cinder-operator/internal/cinder"
	memcachedv1 "github.com/openstack-k8s-operators/infra-operator/apis/memcached/v1beta1"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/pod"
	"github.com/openstack-k8s-operators/lib-common/modules/common/probes"
	"github.com/openstack-k8s-operators/lib-common/modules/users"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// StatefulSet func
func StatefulSet(
	instance *cinderv1.CinderVolume,
	configHash string,
	labels map[string]string,
	annotations map[string]string,
	usesLVM bool,
	topology *topologyv1.Topology,
	memcached *memcachedv1.Memcached,
) (*appsv1.StatefulSet, error) {
	// Both scheme and port are set according to the healthcheck.py script
	scheme := corev1.URISchemeHTTP
	probesPort := int32(8080)

	volumeProbes, err := probes.CreateProbeSet(
		probesPort,
		&scheme,
		instance.Spec.Override.Probes,
		cinder.GetDefaultProbesRPCWorker(cinder.CinderServiceDownTime),
	)
	// Could not process probes config
	if err != nil {
		return nil, err
	}

	cinderVolumeCommand := "/usr/bin/cinder-volume --config-dir /etc/cinder/cinder.conf.d"
	if usesLVM {
		// LVM/iSCSI target creation needs the host network namespace
		cinderVolumeCommand = "nsenter --net=/proc/1/ns/net -- " + cinderVolumeCommand
	}
	args := []string{"-c", cinderVolumeCommand}
	probeCommand := []string{
		"/usr/local/bin/container-scripts/healthcheck.py",
		"volume",
		"/etc/cinder/cinder.conf.d",
	}

	envVars := map[string]env.Setter{}
	envVars["CONFIG_HASH"] = env.SetValue(configHash)

	// Tune glibc for reduced memory usage and fragmentation using single malloc arena for all
	// threads and disabling dynamic thresholds to reduce memory usage when using native threads
	// directly or via eventlet.tpool
	// https://www.gnu.org/software/libc/manual/html_node/Memory-Allocation-Tunables.html
	envVars["MALLOC_ARENA_MAX"] = env.SetValue("1")
	envVars["MALLOC_MMAP_THRESHOLD_"] = env.SetValue("131072")
	envVars["MALLOC_TRIM_THRESHOLD_"] = env.SetValue("262144")

	volumes := GetVolumes(
		cinder.GetOwningCinderName(instance),
		instance.Name,
		instance.Spec.ExtraMounts,
		instance.BackendName(),
	)
	volumeMounts := GetVolumeMounts(
		instance.Spec.ExtraMounts,
		usesLVM,
		instance.BackendName(),
	)

	// Add the CA bundle
	if instance.Spec.TLS.CaBundleSecretName != "" {
		volumes = append(volumes, instance.Spec.TLS.CreateVolume())
		volumeMounts = append(volumeMounts, instance.Spec.TLS.CreateVolumeMounts(nil)...)
	}

	// add MTLS cert if defined
	if memcached.Status.MTLSCert != "" && instance.Spec.MemcachedInstance != nil {
		volumes = append(volumes, memcached.CreateMTLSVolume())
		certMountPath := memcachedv1.CertPathDst
		keyMountPath := memcachedv1.KeyPathDst
		volumeMounts = append(volumeMounts, memcached.CreateMTLSVolumeMounts(&certMountPath, &keyMountPath)...)
	}

	statefulset := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Replicas: instance.Spec.Replicas,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:           instance.Spec.ServiceAccount,
					AutomountServiceAccountToken: ptr.To(false),
					SecurityContext:              pod.RestrictivePodSecurityContext(users.CinderUID, users.CinderGID),
					// Some commands need to be run on the host using nsenter
					// (eg: iscsi commands) so we need to share the PID
					// namespace with the host.
					HostPID: true,
					Containers: []corev1.Container{
						{
							Name: ComponentName,
							Command: []string{
								"/bin/bash",
							},
							Args:  args,
							Image: instance.Spec.ContainerImage,
							// cinder-volume needs Privileged for LVM/iSCSI/multipath
							// device management via the host's nsenter'd binaries
							// (see cinder.RunOnHostVolumeMount) - this can't move to a
							// restrictive/capability-limited SecurityContext.
							SecurityContext: pod.PrivilegedSecurityContext(users.CinderUID, users.CinderGID),
							Env:             env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts:    volumeMounts,
							Resources:       instance.Spec.Resources,
							LivenessProbe:   volumeProbes.Liveness,
							StartupProbe:    volumeProbes.Startup,
						},
						{
							Name:            "probe",
							Command:         probeCommand,
							Image:           instance.Spec.ContainerImage,
							SecurityContext: pod.RestrictiveSecurityContext(users.CinderUID, users.CinderGID),
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if instance.Spec.NodeSelector != nil {
		statefulset.Spec.Template.Spec.NodeSelector = *instance.Spec.NodeSelector
	}

	if topology != nil {
		topology.ApplyTo(&statefulset.Spec.Template)
	} else {
		// If possible two pods of the same service should not
		// run on the same worker node. If this is not possible
		// the get still created on the same worker node.
		statefulset.Spec.Template.Spec.Affinity = cinder.GetPodAffinity(ComponentName)
	}

	return statefulset, nil
}
