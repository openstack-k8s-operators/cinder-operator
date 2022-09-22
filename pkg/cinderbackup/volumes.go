package cinderbackup

import (
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	common "github.com/openstack-k8s-operators/lib-common/modules/common"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(parentName string, name string) []corev1.Volume {
	var config0640AccessMode int32 = 0640

	backupVolumes := []corev1.Volume{
		{
			Name: name + "-config-data",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &config0640AccessMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name + "-config-data",
					},
				},
			},
		},
	}

	return append(cinder.GetVolumes(parentName), backupVolumes...)
}

// GetInitVolumeMounts - Cinder Backup init task VolumeMounts
func GetInitVolumeMounts(name string) []corev1.VolumeMount {

	customConfVolumeMount := corev1.VolumeMount{
		Name:      name + "-config-data",
		MountPath: "/var/lib/config-data/custom/" + common.CustomServiceConfigFileName,
		SubPath:   common.CustomServiceConfigFileName,
		ReadOnly:  true,
	}

	return append(cinder.GetInitVolumeMounts(), customConfVolumeMount)
}

// GetVolumeMounts - Cinder Backup VolumeMounts
func GetVolumeMounts(name string) []corev1.VolumeMount {

	kollaConfigVolumeMount := corev1.VolumeMount{
		Name:      name + "-config-data",
		MountPath: "/var/lib/kolla/config_files/config.json",
		SubPath:   KollaConfig,
		ReadOnly:  true,
	}
	return append(cinder.GetVolumeMounts(), kollaConfigVolumeMount)
}

// GetProbeVolumeMounts - Cinder Backup Probe VolumeMounts
func GetProbeVolumeMounts(name string) []corev1.VolumeMount {

	kollaConfigVolumeMount := corev1.VolumeMount{
		Name:      name + "-config-data",
		MountPath: "/var/lib/kolla/config_files/config.json",
		SubPath:   KollaConfigProbe,
		ReadOnly:  true,
	}
	return append(cinder.GetVolumeMounts(), kollaConfigVolumeMount)
}
