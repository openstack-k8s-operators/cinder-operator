package cinderapi

import (
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

	return append(cinder.GetVolumes(parentName, false, extraVol, cinder.CinderAPIPropagation), volumes...)
}

// GetVolumeMounts - Cinder API VolumeMounts
func GetVolumeMounts(extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-data-custom",
			MountPath: "/etc/cinder/cinder.conf.d",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/var/lib/kolla/config_files/config.json",
			SubPath:   "cinder-api-config.json",
			ReadOnly:  true,
		},
	}

	return append(cinder.GetVolumeMounts(false, extraVol, cinder.CinderAPIPropagation), volumeMounts...)
}

// GetLogVolumeMount - Cinder API LogVolumeMount
func GetLogVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      logVolume,
		MountPath: "/var/log/cinder",
		ReadOnly:  false,
	}
}

// GetLogVolume - Cinder API LogVolume
func GetLogVolume() corev1.Volume {
	return corev1.Volume{
		Name: logVolume,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{Medium: ""},
		},
	}
}
