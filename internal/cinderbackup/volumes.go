package cinderbackup

import (
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/cinder-operator/internal/cinder"
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
	}

	return append(cinder.GetVolumes(parentName, true, extraVol, cinder.CinderBackupPropagation), volumes...)
}

// GetVolumeMounts - Cinder Backup VolumeMounts
func GetVolumeMounts(extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
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

	return append(cinder.GetVolumeMounts(true, extraVol, cinder.CinderBackupPropagation), volumeMounts...)
}
