package cinderapi

import (
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/internal/cinder"
	"github.com/openstack-k8s-operators/lib-common/modules/common/volume"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(parentName string, name string, extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.Volume {
	var configAccessMode int32 = 0440

	volumes := []corev1.Volume{
		{
			Name: "config-data-custom",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &configAccessMode,
					SecretName:  name + "-config-data",
				},
			},
		},
		volume.WritableDirVolume("logs"),
		volume.WritableDirVolume(volume.RunHttpdVolumeName),
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
			MountPath: "/etc/httpd/conf/httpd.conf",
			SubPath:   "httpd.conf",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/httpd/conf.d/ssl.conf",
			SubPath:   "ssl.conf",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/httpd/conf.d/10-cinder_wsgi.conf",
			SubPath:   "10-cinder_wsgi.conf",
			ReadOnly:  true,
		},
		volume.WritableDirVolumeMount(volume.RunHttpdVolumeName, volume.RunHttpdMountPath),
		GetLogVolumeMount(),
	}

	return append(cinder.GetVolumeMounts(false, extraVol, cinder.CinderAPIPropagation), volumeMounts...)
}

// GetLogVolumeMount - Cinder API LogVolumeMount
func GetLogVolumeMount() corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "logs",
		MountPath: "/var/log/cinder",
		ReadOnly:  false,
	}
}
