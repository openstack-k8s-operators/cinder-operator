package common

// GetNodeSelector - returns the NodeSelector
func GetNodeSelector(roleName string) map[string]string {

	// Change nodeSelector
	nodeSelector := "node-role.kubernetes.io/" + roleName
	return map[string]string{nodeSelector: ""}
}
