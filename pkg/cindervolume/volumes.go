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
	var config0644AccessMode int32 = 0644

	volumes := []corev1.Volume{
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
func GetVolumeMounts(name string, extraVol []cinderv1beta1.CinderExtraVolMounts, usesLVM bool) []corev1.VolumeMount {
	var config_data string
	if usesLVM {
		config_data = "cinder-volume-lvm-config.json"
	} else {
		config_data = "cinder-volume-config.json"
	}
	volumeVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-data-custom",
			MountPath: "/etc/cinder/cinder.conf.d",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/var/lib/kolla/config_files/config.json",
			SubPath:   config_data,
			ReadOnly:  true,
		},
	}

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(strings.TrimPrefix(name, "cinder-volume-")))
	return append(cinder.GetVolumeMounts(true, extraVol, propagation), volumeVolumeMounts...)
}
