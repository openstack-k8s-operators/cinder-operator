package cinder

import (
	common "github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/affinity"
	"github.com/openstack-k8s-operators/lib-common/modules/common/probes"

	corev1 "k8s.io/api/core/v1"
	"math"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetOwningCinderName - Given a CinderAPI, CinderScheduler, CinderBackup or CinderVolume
// object, returning the parent Cinder object that created it (if any)
func GetOwningCinderName(instance client.Object) string {
	for _, ownerRef := range instance.GetOwnerReferences() {
		if ownerRef.Kind == "Cinder" {
			return ownerRef.Name
		}
	}

	return ""
}

// GetNetworkAttachmentAddrs - Returns a list of IP addresses for all network attachments.
func GetNetworkAttachmentAddrs(namespace string, networkAttachments []string, networkAttachmentStatus map[string][]string) []string {
	networkAttachmentAddrs := []string{}

	for _, network := range networkAttachments {
		networkName := namespace + "/" + network
		if networkAddrs, ok := networkAttachmentStatus[networkName]; ok {
			networkAttachmentAddrs = append(networkAttachmentAddrs, networkAddrs...)
		}
	}

	return networkAttachmentAddrs
}

// GetPodAffinity - Returns a corev1.Affinity reference for the specified component.
func GetPodAffinity(componentName string) *corev1.Affinity {
	// If possible two pods of the same component (e.g cinder-api) should not
	// run on the same worker node. If this is not possible they get still
	// created on the same worker node.
	return affinity.DistributePods(
		common.ComponentSelector,
		[]string{
			componentName,
		},
		corev1.LabelHostname,
	)
}

// GetDefaultProbesAPI -
func GetDefaultProbesAPI(apiTimeout int) probes.OverrideSpec {
	const failureCount = 3
	period := int32(math.Floor(float64(apiTimeout) / float64(failureCount)))
	// For startup probes, use shorter period for faster startup detection
	startupPeriod := int32(math.Max(5, float64(period)/2))

	// Default values applied to CinderAPI StatefulSets when no
	// overrides are provided
	return probes.OverrideSpec{
		LivenessProbes: &probes.ProbeConf{
			Path:                "/healthcheck",
			TimeoutSeconds:      5,
			PeriodSeconds:       period,
			InitialDelaySeconds: 5,
		},
		ReadinessProbes: &probes.ProbeConf{
			Path:                "/healthcheck",
			TimeoutSeconds:      5,
			PeriodSeconds:       period,
			InitialDelaySeconds: 5,
		},
		StartupProbes: &probes.ProbeConf{
			TimeoutSeconds:      5,
			PeriodSeconds:       startupPeriod,
			InitialDelaySeconds: 5,
			FailureThreshold:    12,
		},
	}
}

// GetDefaultProbesRPCWorker returns default probe configuration for RPC worker
// processes (cinder-volume, cinder-scheduler, cinder-backup). These processes
// expose no HTTP healthcheck endpoint, so Path is intentionally omitted.
// A dedicated serviceDownTime field is not currently exposed in the operator's
// API, and we rely on the default provided by cinder to compute the default
// values.
// This is an established value, and dedicated probes tuning can be performed
// via the dedicated interface.
// https://opendev.org/openstack/cinder/src/branch/master/cinder/common/config.py#L121
func GetDefaultProbesRPCWorker(serviceDownTime int) probes.OverrideSpec {
	const failureCount = 3
	period := int32(math.Floor(float64(serviceDownTime) / float64(failureCount)))
	// For startup probes, use shorter period for faster startup detection
	startupPeriod := int32(math.Max(5, float64(period)/2))

	return probes.OverrideSpec{
		LivenessProbes: &probes.ProbeConf{
			TimeoutSeconds:      5,
			PeriodSeconds:       period,
			InitialDelaySeconds: 15,
		},
		ReadinessProbes: &probes.ProbeConf{
			TimeoutSeconds:      5,
			PeriodSeconds:       period,
			InitialDelaySeconds: 15,
		},
		StartupProbes: &probes.ProbeConf{
			TimeoutSeconds:      5,
			PeriodSeconds:       startupPeriod,
			InitialDelaySeconds: period,
			FailureThreshold:    12,
		},
	}
}
