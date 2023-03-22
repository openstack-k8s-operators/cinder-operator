package cinderapi

import (
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/pkg/cinder"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(parentName string, name string, secretNames []string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.Volume {
	var config0640AccessMode int32 = 0640

	volumes := []corev1.Volume{
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

	return append(cinder.GetVolumes(parentName, false, extraVol, cinder.CinderAPIPropagation), volumes...)
}

// GetInitVolumeMounts - Cinder API init task VolumeMounts
func GetInitVolumeMounts(secretNames []string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {

	initVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-data-custom",
			MountPath: "/var/lib/config-data/custom",
			ReadOnly:  true,
		},
	}

	initVolumeMounts = append(initVolumeMounts, cinder.GetSecretVolumeMounts(secretNames)...)

	return append(cinder.GetInitVolumeMounts(extraVol, cinder.CinderAPIPropagation), initVolumeMounts...)
}

// GetVolumeMounts - Cinder API VolumeMounts
func GetVolumeMounts(extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {
	return cinder.GetVolumeMounts(false, extraVol, cinder.CinderAPIPropagation)
}
