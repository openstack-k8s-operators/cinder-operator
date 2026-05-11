package cinder

import (
	"fmt"
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ManageJobType defines the type of cinder-manage command to run
type ManageJobType string

const (
	// DBSyncCommand - direct cinder-manage command without kolla wrapper
	// TODO: Once we work on update/upgrades we'll need to use the
	//       --bump-versions flag conditionally (only for adoption) and do
	//       the restart cycle of services as described in the upstream
	//       rolling upgrades process.
	DBSyncCommand = "/usr/bin/cinder-manage --config-dir /etc/cinder/cinder.conf.d db sync --bump-versions"
	// OnlineDataMigrationsCommand - for running online data migrations during upgrades
	OnlineDataMigrationsCommand = "/usr/bin/cinder-manage --config-dir /etc/cinder/cinder.conf.d db online_data_migrations"
	// DbSyncJobType for database synchronization
	DbSyncJobType ManageJobType = "db-sync"
	// OnlineDataMigrationsJobType for online data migrations
	OnlineDataMigrationsJobType ManageJobType = "online-data-migrations"
)

// ManageJob - creates a job for running various cinder-manage commands
func ManageJob(
	instance *cinderv1beta1.Cinder,
	labels map[string]string,
	annotations map[string]string,
	jobNameSuffix ManageJobType,
	command string,
) *batchv1.Job {
	var config0644AccessMode int32 = 0644

	// Unlike the individual cinder services, cinder-manage jobs don't need a
	// secret that contains all of the config snippets required by every
	// service, The two snippet files that it does need (DefaultsConfigFileName
	// and CustomConfigFileName) can be extracted from the top-level cinder
	// config-data secret.
	jobVolume := []corev1.Volume{
		{
			Name: "config-data",
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

	jobMounts := []corev1.VolumeMount{
		{
			Name:      "config-data",
			MountPath: "/etc/cinder/cinder.conf.d",
			ReadOnly:  true,
		},
	}

	// add CA cert if defined
	if instance.Spec.CinderAPI.TLS.CaBundleSecretName != "" {
		jobVolume = append(jobVolume, instance.Spec.CinderAPI.TLS.CreateVolume())
		jobMounts = append(jobMounts, instance.Spec.CinderAPI.TLS.CreateVolumeMounts(nil)...)
	}

	args := []string{"-c", command}

	runAsUser := int64(0)
	envVars := map[string]env.Setter{}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", instance.Name, string(jobNameSuffix)),
			Namespace: instance.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					ServiceAccountName: instance.RbacResourceName(),
					Containers: []corev1.Container{
						{
							Name: fmt.Sprintf("%s-%s", instance.Name, string(jobNameSuffix)),
							Command: []string{
								"/bin/bash",
							},
							Args:  args,
							Image: instance.Spec.CinderAPI.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &runAsUser,
							},
							Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: jobMounts,
						},
					},
					Volumes: jobVolume,
				},
			},
		},
	}

	if instance.Spec.NodeSelector != nil {
		job.Spec.Template.Spec.NodeSelector = *instance.Spec.NodeSelector
	}

	return job
}

// DbSyncJob func - backward compatible wrapper for database sync
func DbSyncJob(instance *cinderv1beta1.Cinder, labels map[string]string, annotations map[string]string) *batchv1.Job {
	return ManageJob(instance, labels, annotations, DbSyncJobType, DBSyncCommand)
}

// OnlineDataMigrationsJob creates a job for running cinder-manage db online_data_migrations
func OnlineDataMigrationsJob(instance *cinderv1beta1.Cinder, labels map[string]string, annotations map[string]string) *batchv1.Job {
	return ManageJob(instance, labels, annotations, OnlineDataMigrationsJobType, OnlineDataMigrationsCommand)
}
