package cindervolume

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	"strings"

	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(parentName string, name string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.Volume {
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

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(strings.TrimPrefix(name, "cinder-volume-")))
	return append(cinder.GetVolumes(parentName, true, extraVol, propagation), volumeVolumes...)
}

// GetInitVolumeMounts - Cinder Volume init task VolumeMounts
func GetInitVolumeMounts(name string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {

	customConfVolumeMount := corev1.VolumeMount{
		Name:      "config-data-custom",
		MountPath: "/var/lib/config-data/custom",
		ReadOnly:  true,
	}

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(strings.TrimPrefix(name, "cinder-volume-")))
	return append(cinder.GetInitVolumeMounts(extraVol, propagation), customConfVolumeMount)
}

// GetVolumeMounts - Cinder Volume VolumeMounts
func GetVolumeMounts(name string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {
	volumeVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "var-lib-cinder",
			MountPath: "/var/lib/cinder",
		},
	}

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(strings.TrimPrefix(name, "cinder-volume-")))
	return append(cinder.GetVolumeMounts(true, extraVol, propagation), volumeVolumeMounts...)
}
