package cindervolume

import (
	corev1 "k8s.io/api/core/v1"
)

// GetVolumes - common Volumes used by many service pods
func GetVolumes(name string) []corev1.Volume {
	var dirOrCreate = corev1.HostPathDirectoryOrCreate

	return []corev1.Volume{
		{
			Name: "etc-iscsi",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/iscsi",
				},
			},
		},
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
		{
			Name: "sys",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/sys",
				},
			},
		},
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
			Name: "var-lib-iscsi",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/iscsi",
					Type: &dirOrCreate,
				},
			},
		},
		{
			Name: "volume-config-data-custom",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name + "-config-data-custom",
					},
				},
			},
		},
	}

}

// GetInitVolumeMounts - Nova Control Plane init task VolumeMounts
func GetInitVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "volume-config-data-custom",
			MountPath: "/var/lib/config-data/volume-custom",
			ReadOnly:  true,
		},
	}

}

// GetVolumeMounts - common VolumeMounts
func GetVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
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
			Name:      "var-lib-cinder",
			MountPath: "/var/lib/cinder",
		},
		{
			Name:      "var-lib-iscsi",
			MountPath: "/var/lib/iscsi",
		},
	}

}
