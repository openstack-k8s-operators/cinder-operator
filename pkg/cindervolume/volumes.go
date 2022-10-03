package cindervolume

import (
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(parentName string, name string) []corev1.Volume {
	var config0640AccessMode int32 = 0640
	var dirOrCreate = corev1.HostPathDirectoryOrCreate

	volumeVolumes := []corev1.Volume{
		{
			Name: "var-lib-cinder",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/cinder",
					Type: &dirOrCreate,
				},
			},
		},
		{
			Name: "config-data-custom",
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

	return append(cinder.GetVolumes(parentName, true), volumeVolumes...)
}

// GetInitVolumeMounts - Cinder Volume init task VolumeMounts
func GetInitVolumeMounts() []corev1.VolumeMount {

	customConfVolumeMount := corev1.VolumeMount{
		Name:      "config-data-custom",
		MountPath: "/var/lib/config-data/custom",
		ReadOnly:  true,
	}

	return append(cinder.GetInitVolumeMounts(), customConfVolumeMount)
}

// GetVolumeMounts - Cinder Volume VolumeMounts
func GetVolumeMounts() []corev1.VolumeMount {
	volumeVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "var-lib-cinder",
			MountPath: "/var/lib/cinder",
		},
	}

	return append(cinder.GetVolumeMounts(true), volumeVolumeMounts...)
}
