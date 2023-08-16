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
	var config0644AccessMode int32 = 0644
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
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &config0644AccessMode,
					SecretName:  name + "-config-data",
				},
			},
		},
	}

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(strings.TrimPrefix(name, "cinder-volume-")))
	return append(cinder.GetVolumes(parentName, true, extraVol, propagation), volumes...)
}

// GetVolumeMounts - Cinder Volume VolumeMounts
func GetVolumeMounts(name string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {
	volumeVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-data-custom",
			MountPath: "/etc/cinder/cinder.conf.d",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/var/lib/kolla/config_files/config.json",
			SubPath:   "cinder-volume-config.json",
			ReadOnly:  true,
		},
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
