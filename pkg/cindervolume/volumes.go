package cindervolume

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	"strings"

	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(parentName string, name string, secretNames []string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.Volume {
	var config0640AccessMode int32 = 0640
	var dirOrCreate = corev1.HostPathDirectoryOrCreate

	volumes := []corev1.Volume{
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
			Name: "etc-nvme",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/nvme",
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

	volumes = append(volumes, cinder.GetSecretVolumes(secretNames)...)

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(strings.TrimPrefix(name, "cinder-volume-")))
	return append(cinder.GetVolumes(parentName, true, extraVol, propagation), volumes...)
}

// GetInitVolumeMounts - Cinder Volume init task VolumeMounts
func GetInitVolumeMounts(name string, secretNames []string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {
	initVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-data-custom",
			MountPath: "/var/lib/config-data/custom",
			ReadOnly:  true,
		},
	}

	initVolumeMounts = append(initVolumeMounts, cinder.GetSecretVolumeMounts(secretNames)...)

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(strings.TrimPrefix(name, "cinder-volume-")))
	return append(cinder.GetInitVolumeMounts(extraVol, propagation), initVolumeMounts...)
}

// GetVolumeMounts - Cinder Volume VolumeMounts
func GetVolumeMounts(name string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {
	volumeVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "var-lib-cinder",
			MountPath: "/var/lib/cinder",
		},
		{
			Name:      "etc-nvme",
			MountPath: "/etc/nvme",
		},
	}

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(strings.TrimPrefix(name, "cinder-volume-")))
	return append(cinder.GetVolumeMounts(true, extraVol, propagation), volumeVolumeMounts...)
}
