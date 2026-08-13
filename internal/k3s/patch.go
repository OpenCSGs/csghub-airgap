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

	// Pass values as separate exec args rather than building a shell command
	// string, so user-supplied domain/ip can't inject shell commands.
	if err = shell.Execute("env",
		fmt.Sprintf("DOMAIN=%s", domain),
		fmt.Sprintf("TARGET_IP=%s", ip),
		"bash", patchScript,
	); err != nil {
		return err
	}

	return nil
}
