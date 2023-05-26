package cinder

import (
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	common "github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DBSyncCommand -
	DBSyncCommand = "/usr/local/bin/kolla_set_configs && /usr/local/bin/kolla_start"
)

// DbSyncJob func
func DbSyncJob(instance *cinderv1beta1.Cinder, labels map[string]string, annotations map[string]string) *batchv1.Job {
	var config0644AccessMode int32 = 0644

	// Unlike the individual cinder services, the DbSyncJob doesn't need a
	// secret that contains all of the config snippets required by every
	// service, The two snippet files that it does need (DefaultsConfigFileName
	// and CustomConfigFileName) can be extracted from the top-level cinder
	// config-data secret.
	dbSyncVolume := corev1.Volume{
		Name: "db-sync-config-data",
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
	}

	dbSyncMount := corev1.VolumeMount{
		Name:      "db-sync-config-data",
		MountPath: "/etc/cinder/cinder.conf.d",
		ReadOnly:  true,
	}

	dbSyncExtraMounts := []cinderv1beta1.CinderExtraVolMounts{}

	args := []string{"-c"}
	if instance.Spec.Debug.DBSync {
		args = append(args, common.DebugCommand)
	} else {
		args = append(args, DBSyncCommand)
	}

	runAsUser := int64(0)
	envVars := map[string]env.Setter{}
	envVars["KOLLA_CONFIG_FILE"] = env.SetValue(KollaConfigDbSync)
	envVars["KOLLA_CONFIG_STRATEGY"] = env.SetValue("COPY_ALWAYS")
	envVars["KOLLA_BOOTSTRAP"] = env.SetValue("TRUE")

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-db-sync",
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
							Name: instance.Name + "-db-sync",
							Command: []string{
								"/bin/bash",
							},
							Args:  args,
							Image: instance.Spec.CinderAPI.ContainerImage,
							SecurityContext: &corev1.SecurityContext{
								RunAsUser: &runAsUser,
							},
							Env:          env.MergeEnvs([]corev1.EnvVar{}, envVars),
							VolumeMounts: append(GetVolumeMounts(false, dbSyncExtraMounts, DbsyncPropagation), dbSyncMount),
						},
					},
					Volumes: append(GetVolumes(instance.Name, false, dbSyncExtraMounts, DbsyncPropagation), dbSyncVolume),
				},
			},
		},
	}

	initContainerDetails := APIDetails{
		ContainerImage:       instance.Spec.CinderAPI.ContainerImage,
		DatabaseHost:         instance.Status.DatabaseHostname,
		DatabaseUser:         instance.Spec.DatabaseUser,
		DatabaseName:         DatabaseName,
		OSPSecret:            instance.Spec.Secret,
		DBPasswordSelector:   instance.Spec.PasswordSelectors.Database,
		UserPasswordSelector: instance.Spec.PasswordSelectors.Service,
		VolumeMounts:         GetInitVolumeMounts(dbSyncExtraMounts, DbsyncPropagation),
		Debug:                instance.Spec.Debug.DBInitContainer,
	}
	job.Spec.Template.Spec.InitContainers = InitContainer(initContainerDetails)

	return job
}
