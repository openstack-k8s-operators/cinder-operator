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

package cinderscheduler

import (
	cinderv1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	cinder "github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	common "github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/affinity"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// ServiceCommand -
	ServiceCommand = "/usr/local/bin/kolla_set_configs && /usr/local/bin/kolla_start"
)

// StatefulSet func
func StatefulSet(
	instance *cinderv1.CinderScheduler,
	configHash string,
	labels map[string]string,
	annotations map[string]string,
) *appsv1.StatefulSet {
	rootUser := int64(0)
	cinderUser := int64(cinderv1.CinderUserID)
	cinderGroup := int64(cinderv1.CinderGroupID)

	// TODO until we determine how to properly query for these
	livenessProbe := &corev1.Probe{
		// TODO might need tuning
		TimeoutSeconds:      5,
		PeriodSeconds:       3,
		InitialDelaySeconds: 3,
	}

	startupProbe := &corev1.Probe{
		TimeoutSeconds:      5,
		FailureThreshold:    12,
		PeriodSeconds:       5,
		InitialDelaySeconds: 5,
	}

	args := []string{"-c"}
	var probeCommand []string
	// When debugging the service container will run kolla_set_configs and
	// sleep forever and the probe container will just sleep forever.
	if instance.Spec.Debug.Service {
		args = append(args, common.DebugCommand)
		livenessProbe.Exec = &corev1.ExecAction{
			Command: []string{
				"/bin/true",
			},
		}
		startupProbe.Exec = livenessProbe.Exec
		probeCommand = []string{
			"/bin/sleep", "infinity",
		}
	} else {
		args = append(args, ServiceCommand)
		livenessProbe.HTTPGet = &corev1.HTTPGetAction{
			Port: intstr.FromInt(8080),
		}
		startupProbe.HTTPGet = livenessProbe.HTTPGet
		probeCommand = []string{
			"/usr/local/bin/container-scripts/healthcheck.py",
			"scheduler",
			"/etc/cinder/cinder.conf.d",
		}
	}

	envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["CONFIG_HASH"] = env.SetValue(configHash)

	volumes := GetVolumes(
		cinder.GetOwningCinderName(instance),
		instance.Name,
		instance.Spec.ExtraMounts)
	volumeMounts := GetVolumeMounts(instance.Spec.ExtraMounts)

	// Add the CA bundle
	if instance.Spec.TLS.CaBundleSecretName != "" {
		volumes = append(volumes, instance.Spec.TLS.CreateVolume())
		volumeMounts = append(volumeMounts, instance.Spec.TLS.CreateVolumeMounts(nil)...)
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
					ServiceAccountName: instance.Spec.ServiceAccount,
					Containers: []corev1.Container{
						{
							Name: cinder.ServiceName + "-scheduler",
							Command: []string{
								"/bin/bash",
							},
							Args:  args,
							Image: instance.Spec.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &rootUser,
							},
							Env:           env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts:  volumeMounts,
							Resources:     instance.Spec.Resources,
							LivenessProbe: livenessProbe,
							StartupProbe:  startupProbe,
						},
						{
							Name:    "probe",
							Command: probeCommand,
							Image:   instance.Spec.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &cinderUser,
								RunAsGroup: &cinderGroup,
							},
							VolumeMounts: volumeMounts,
						},
					},
					NodeSelector: instance.Spec.NodeSelector,
					Volumes:      volumes,
				},
			},
		},
	}

	// If possible two pods of the same service should not
	// run on the same worker node. If this is not possible
	// the get still created on the same worker node.
	statefulset.Spec.Template.Spec.Affinity = affinity.DistributePods(
		common.AppSelector,
		[]string{
			cinder.ServiceName,
		},
		corev1.LabelHostname,
	)
	if instance.Spec.NodeSelector != nil && len(instance.Spec.NodeSelector) > 0 {
		statefulset.Spec.Template.Spec.NodeSelector = instance.Spec.NodeSelector
	}

	return statefulset
}
