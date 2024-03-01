package cinder

import (
	common "github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/affinity"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"

	corev1 "k8s.io/api/core/v1"
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

// AddNodeSelectorHash - Adds a hash of a nodeSelector map to the envVars.
func AddNodeSelectorHash(nodeSelector map[string]string, envVars *map[string]env.Setter) error {
	hash, err := util.ObjectHash(nodeSelector)
	if err != nil {
		(*envVars)["NodeSelectorHash"] = env.SetValue(hash)
	}
	return err
}
