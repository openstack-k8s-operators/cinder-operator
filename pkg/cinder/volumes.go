package cinder

import (
	"strconv"

	cinderv1beta1 "github.com/openstack-k8s-operators/cinder-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/storage"
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes -
func GetVolumes(name string, storageSvc bool, extraVol []cinderv1beta1.CinderExtraVolMounts, svc []storage.PropagationType) []corev1.Volume {
	var scriptsVolumeDefaultMode int32 = 0755
	var config0640AccessMode int32 = 0640
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
			Name: "scripts",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &scriptsVolumeDefaultMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name + "-scripts",
					},
				},
			},
		},
		{
			Name: "config-data",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: &config0640AccessMode,
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name + "-config-data",
					},
				},
			},
		},
		{
			Name: "config-data-merged",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{Medium: ""},
			},
		},
	}

	// Volume and backup services require extra directories
	if storageSvc {
		storageVolumes := []corev1.Volume{
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
			res = append(res, vol.Volumes...)
		}
	}
	return res
}

// GetInitVolumeMounts - Nova Control Plane init task VolumeMounts
func GetInitVolumeMounts(extraVol []cinderv1beta1.CinderExtraVolMounts, svc []storage.PropagationType) []corev1.VolumeMount {
	vm := []corev1.VolumeMount{
		{
			Name:      "scripts",
			MountPath: "/usr/local/bin/container-scripts",
			ReadOnly:  true,
		},
		{
			Name:      "config-data",
			MountPath: "/var/lib/config-data/default",
			ReadOnly:  true,
		},
		{
			Name:      "config-data-merged",
			MountPath: "/var/lib/config-data/merged",
			ReadOnly:  false,
		},
	}

	for _, exv := range extraVol {
		for _, vol := range exv.Propagate(svc) {
			vm = append(vm, vol.Mounts...)
		}
	}
	return vm
}

// GetVolumeMounts - Nova Control Plane VolumeMounts
func GetVolumeMounts(storageSvc bool, extraVol []cinderv1beta1.CinderExtraVolMounts, svc []storage.PropagationType) []corev1.VolumeMount {
	res := []corev1.VolumeMount{
		{
			Name:      "etc-machine-id",
			MountPath: "/etc/machine-id",
			ReadOnly:  true,
		},
		{
			Name:      "scripts",
			MountPath: "/usr/local/bin/container-scripts",
			ReadOnly:  true,
		},
		{
			Name:      "config-data-merged",
			MountPath: "/var/lib/config-data/merged",
			ReadOnly:  false,
		},
	}

	// Volume and backup services require extra directories
	if storageSvc {
		storageVolumeMounts := []corev1.VolumeMount{
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

// GetSecretVolumes - Returns a list of volumes associated with a list of Secret names
func GetSecretVolumes(secretNames []string) []corev1.Volume {
	var config0640AccessMode int32 = 0640
	secretVolumes := []corev1.Volume{}

	for _, secretName := range secretNames {
		secretVol := corev1.Volume{
			Name: secretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  secretName,
					DefaultMode: &config0640AccessMode,
				},
			},
		}
		secretVolumes = append(secretVolumes, secretVol)
	}
	return secretVolumes
}

// GetSecretVolumeMounts - Returns a list of volume mounts associated with a list of Secret names
func GetSecretVolumeMounts(secretNames []string) []corev1.VolumeMount {
	secretMounts := []corev1.VolumeMount{}

	for idx, secretName := range secretNames {
		secretMount := corev1.VolumeMount{
			Name: secretName,
			// Each secret needs its own MountPath
			MountPath: "/var/lib/config-data/secret-" + strconv.Itoa(idx),
			ReadOnly:  true,
		}
		secretMounts = append(secretMounts, secretMount)
	}
	return secretMounts
}
