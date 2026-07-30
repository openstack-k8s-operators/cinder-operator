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

// runOnHostVolumeMount returns a VolumeMount that shims a host storage binary
// via the "scripts" secret's run-on-host nsenter wrapper, so cinder-backup can
// invoke host-installed multipath/iscsi tooling from inside the container
// (the pod already runs with HostPID: true).
func runOnHostVolumeMount(destPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      "scripts",
		MountPath: destPath,
		SubPath:   "run-on-host",
	}
}

// GetVolumeMounts - Cinder Backup VolumeMounts
func GetVolumeMounts(extraVol []cinderv1beta1.CinderExtraVolMounts) []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "config-data-custom",
			MountPath: "/etc/cinder/cinder.conf.d",
			ReadOnly:  true,
		},
		runOnHostVolumeMount("/usr/sbin/multipath"),
		runOnHostVolumeMount("/usr/sbin/multipathd"),
		runOnHostVolumeMount("/usr/sbin/iscsiadm"),
		runOnHostVolumeMount("/lib/udev/scsi_id"),
		runOnHostVolumeMount("/usr/sbin/cryptsetup"),
		runOnHostVolumeMount("/usr/sbin/nvme"),
	}

	return append(cinder.GetVolumeMounts(true, extraVol, cinder.CinderBackupPropagation), volumeMounts...)
}
