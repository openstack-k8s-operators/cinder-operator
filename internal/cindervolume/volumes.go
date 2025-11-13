package cindervolume

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"

	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/internal/cinder"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(parentName string, name string, extraVol []cinderv1beta1.CinderExtraVolMounts, propagationInstanceName string) []corev1.Volume {
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
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(propagationInstanceName))
	return append(cinder.GetVolumes(parentName, true, extraVol, propagation), volumes...)
}

// GetVolumeMounts - Cinder Volume VolumeMounts
func GetVolumeMounts(extraVol []cinderv1beta1.CinderExtraVolMounts, usesLVM bool, propagationInstanceName string) []corev1.VolumeMount {
	var configData string
	if usesLVM {
		configData = "cinder-volume-lvm-config.json"
	} else {
		configData = "cinder-volume-config.json"
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
			SubPath:   configData,
			ReadOnly:  true,
		},
	}

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(propagationInstanceName))
	return append(cinder.GetVolumeMounts(true, extraVol, propagation), volumeVolumeMounts...)
}
