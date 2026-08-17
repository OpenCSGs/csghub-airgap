package chart

import (
	"fmt"
	"os"
)

// DiscoverKubeConfig returns the first reachable kubeconfig path.
// Kubeconfig is needed to talk to a Kubernetes API server (used by helm
// install/upgrade/uninstall/list). Try k3s first, then standard locations
// so that deploy --skip-k3s (which uses a stock k8s) still works.
func DiscoverKubeConfig() (string, error) {
	if env := os.Getenv("KUBECONFIG"); env != "" {
		return env, nil
	}
	candidates := []string{
		"/etc/rancher/k3s/k3s.yaml",
		"/root/.kube/config",
		"/etc/kubernetes/admin.conf",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no kubeconfig found (tried %s and KUBECONFIG env)", candidates)
}