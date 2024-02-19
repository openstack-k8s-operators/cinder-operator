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
	cinderv1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"

	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

// CronJob func
func CronJob(
	instance *cinderv1.Cinder,
	labels map[string]string,
	annotations map[string]string,
) *batchv1.CronJob {
	cinderUser := int64(cinderv1.CinderUserID)
	cinderGroup := int64(cinderv1.CinderGroupID)
	config0644AccessMode := int32(0644)

	debugArg := ""
	if instance.Spec.Debug.DBPurge {
		debugArg = " --debug"
	}

	dbPurgeCommand := fmt.Sprintf(
		"/usr/bin/cinder-manage%s --config-dir /etc/cinder/cinder.conf.d db purge %d",
		debugArg,
		instance.Spec.DBPurge.Age)

	args := []string{"-c", dbPurgeCommand}

	cronJobVolumes := []corev1.Volume{
		{
			Name: "db-purge-config-data",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &config0644AccessMode,
					SecretName:  instance.Name + "-config-data",
					Items: []corev1.KeyToPath{
						{
							Key:  DefaultsConfigFileName,
							Path: DefaultsConfigFileName,
						},
						{
							Key:  CustomConfigFileName,
							Path: CustomConfigFileName,
						},
					},
				},
			},
		},
	}
	cronJobVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "db-purge-config-data",
			MountPath: "/etc/cinder/cinder.conf.d",
			ReadOnly:  true,
		},
	}

	// add CA cert if defined
	if instance.Spec.CinderAPI.TLS.CaBundleSecretName != "" {
		cronJobVolumes = append(cronJobVolumes, instance.Spec.CinderAPI.TLS.CreateVolume())
		cronJobVolumeMounts = append(cronJobVolumeMounts, instance.Spec.CinderAPI.TLS.CreateVolumeMounts(nil)...)
	}

	cronjob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-db-purge", ServiceName),
			Namespace:   instance.Namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          instance.Spec.DBPurge.Schedule,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      labels,
				},
				Spec: batchv1.JobSpec{
					Parallelism: ptr.To(int32(1)),
					Completions: ptr.To(int32(1)),
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: annotations,
							Labels:      labels,
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  ServiceName + "-db-purge",
									Image: instance.Spec.CinderAPI.ContainerImage,
									Command: []string{
										"/bin/bash",
									},
									Args:         args,
									VolumeMounts: cronJobVolumeMounts,
									SecurityContext: &corev1.SecurityContext{
										RunAsUser:  &cinderUser,
										RunAsGroup: &cinderGroup,
									},
								},
							},
							Volumes:            cronJobVolumes,
							RestartPolicy:      corev1.RestartPolicyNever,
							ServiceAccountName: instance.RbacResourceName(),
						},
					},
				},
			},
		},
	}
	if instance.Spec.NodeSelector != nil && len(instance.Spec.NodeSelector) > 0 {
		cronjob.Spec.JobTemplate.Spec.Template.Spec.NodeSelector = instance.Spec.NodeSelector
	}
	return cronjob
}
