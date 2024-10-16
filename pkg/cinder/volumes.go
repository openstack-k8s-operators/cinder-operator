package cinder

import (
	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(name string, storageSvc bool, extraVol []cinderv1beta1.CinderExtraVolMounts, svc []storage.PropagationType) []corev1.Volume {
	var scriptsVolumeDefaultMode int32 = 0755
	var config0644AccessMode int32 = 0644
	var dirOrCreate = corev1.HostPathDirectoryOrCreate

	res := []corev1.Volume{
		{
			Name: "etc-machine-id",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/machine-id",
				},
			},
		},
		{
			Name: "etc-localtime",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/localtime",
				},
			},
		},
		{
			Name: "scripts",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &scriptsVolumeDefaultMode,
					SecretName:  name + "-scripts",
				},
			},
		},
		{
			Name: "config-data",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					DefaultMode: &config0644AccessMode,
					SecretName:  name + "-config-data",
				},
			},
		},
	}

	// Volume and backup services require extra directories
	if storageSvc {
		storageVolumes := []corev1.Volume{
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
			// os-brick reads the initiatorname.iscsi from theere
			{
				Name: "etc-iscsi",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/etc/iscsi",
					},
				},
			},
			// /dev needed for os-brick code that looks for things there and
			// for Volume and Backup operations that access data
			{
				Name: "dev",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/dev",
					},
				},
			},
			{
				Name: "lib-modules",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/lib/modules",
					},
				},
			},
			{
				Name: "run",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/run",
					},
				},
			},
			// /sys needed for os-brick code that looks for information there
			{
				Name: "sys",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/sys",
					},
				},
			},
			// os-brick locks need to be shared between the different volume
			// consumers (available in OSP18)
			{
				Name: "var-locks-brick",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/locks/openstack/os-brick",
						Type: &dirOrCreate,
					},
				},
			},
			// In OSP17 there is no os-brick specific lock path conf option,
			// so the global cinder one is used for os-brick locks
			{
				Name: "var-locks-cinder",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: "/var/locks/openstack/cinder",
						Type: &dirOrCreate,
					},
				},
			},
		}

		res = append(res, storageVolumes...)
	}

	for _, exv := range extraVol {
		for _, vol := range exv.Propagate(svc) {
			for _, v := range vol.Volumes {
				convertedVolume := corev1.Volume{
					Name:         v.Name,
					VolumeSource: storage.ConvertVolumeSource(&v.VolumeSource),
				}
				res = append(res, convertedVolume)
			}
		}
	}
	return res
}

// GetVolumeMounts - Cinder Control Plane VolumeMounts
func GetVolumeMounts(storageSvc bool, extraVol []cinderv1beta1.CinderExtraVolMounts, svc []storage.PropagationType) []corev1.VolumeMount {
	res := []corev1.VolumeMount{
		{
			Name:      "etc-machine-id",
			MountPath: "/etc/machine-id",
			ReadOnly:  true,
		},
		{
			Name:      "etc-localtime",
			MountPath: "/etc/localtime",
			ReadOnly:  true,
		},
		{
			Name:      "scripts",
			MountPath: "/usr/local/bin/container-scripts",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/var/lib/config-data/merged",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/etc/my.cnf",
			SubPath:   MyCnfFileName,
			ReadOnly:  true,
		},
	}

	// Volume and backup services require extra directories
	if storageSvc {
		storageVolumeMounts := []corev1.VolumeMount{
			{
				Name:      "var-lib-cinder",
				MountPath: "/var/lib/cinder",
			},
			{
				Name:      "etc-nvme",
				MountPath: "/etc/nvme",
			},
			{
				Name:      "etc-iscsi",
				MountPath: "/etc/iscsi",
				ReadOnly:  true,
			},
			{
				Name:      "dev",
				MountPath: "/dev",
			},
			{
				Name:      "lib-modules",
				MountPath: "/lib/modules",
				ReadOnly:  true,
			},
			{
				Name:      "run",
				MountPath: "/run",
			},
			{
				Name:      "sys",
				MountPath: "/sys",
			},
			{
				Name:      "var-locks-brick",
				MountPath: "/var/locks/openstack/os-brick",
				ReadOnly:  false,
			},
			{
				Name:      "var-locks-cinder",
				MountPath: "/var/locks/openstack/cinder",
				ReadOnly:  false,
			},
		}
		res = append(res, storageVolumeMounts...)
	}

	for _, exv := range extraVol {
		for _, vol := range exv.Propagate(svc) {
			res = append(res, vol.Mounts...)
		}
	}
	return res
}
