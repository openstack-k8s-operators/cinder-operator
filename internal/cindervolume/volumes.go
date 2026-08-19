package cindervolume

import (
	"github.com/openstack-k8s-operators/lib-common/modules/storage"

	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/internal/cinder"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(parentName string, name string, extraVol []cinderv1beta1.CinderExtraVolMounts, propagationInstanceName string) []corev1.Volume {
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
	}

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(propagationInstanceName))
	return append(cinder.GetVolumes(parentName, true, extraVol, propagation), volumes...)
}

// GetVolumeMounts - Cinder Volume VolumeMounts
func GetVolumeMounts(extraVol []cinderv1beta1.CinderExtraVolMounts, usesLVM bool, propagationInstanceName string) []corev1.VolumeMount {
	volumeVolumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-data-custom",
			MountPath: "/etc/cinder/cinder.conf.d",
			ReadOnly:  true,
		},
		cinder.RunOnHostVolumeMount("/usr/sbin/multipath"),
		cinder.RunOnHostVolumeMount("/usr/sbin/multipathd"),
		cinder.RunOnHostVolumeMount("/usr/sbin/iscsiadm"),
		cinder.RunOnHostVolumeMount("/lib/udev/scsi_id"),
		cinder.RunOnHostVolumeMount("/usr/sbin/cryptsetup"),
		cinder.RunOnHostVolumeMount("/usr/sbin/nvme"),
	}
	if usesLVM {
		volumeVolumeMounts = append(volumeVolumeMounts, cinder.RunOnHostVolumeMount("/usr/sbin/lvm"))
	}

	// Set the propagation levels for CinderVolume, including the backend name
	propagation := append(cinder.CinderVolumePropagation, storage.PropagationType(propagationInstanceName))
	return append(cinder.GetVolumeMounts(true, extraVol, propagation), volumeVolumeMounts...)
}
