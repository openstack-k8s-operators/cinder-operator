package cinder

import "sigs.k8s.io/controller-runtime/pkg/client"

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
