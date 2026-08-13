package k3s

import (
	"csghub-airgap/internal/infra/net"
	"csghub-airgap/internal/infra/shell"
	"csghub-airgap/internal/spec"
	"fmt"
	"path/filepath"
)

func PatchInstall(ks *spec.K3sSpec, domain string) error {
	patchScript := filepath.Join(filepath.Dir(ks.ScriptPath()), "k3s-patch.sh")

	defaultIfaceName, err := net.GetDefaultInterface()
	if err != nil {
		return err
	}

	ip, err := net.GetInterfaceIP(defaultIfaceName)
	if err != nil {
		return err
	}

	// Pass values as env vars with quoting rather than interpolating into the
	// command string, so user-supplied domain/ip can't inject shell commands.
	cmd := fmt.Sprintf("env DOMAIN=%q TARGET_IP=%q bash %s", domain, ip, patchScript)
	if err = shell.Execute("/bin/bash", "-c", cmd); err != nil {
		return err
	}

	return nil
}
