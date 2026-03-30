package k3s

import (
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"fmt"
	"path/filepath"
)

func PatchInstall(ks *spec.K3sSpec, domain string) error {
	patchScript := filepath.Join(filepath.Dir(ks.ScriptPath()), "k3s-patch.sh")

	defaultIfaceName, err := utils.GetDefaultInterface()
	if err != nil {
		return err
	}

	ip, err := utils.GetInterfaceIP(defaultIfaceName)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("DOMAIN=%s TARGET_IP=%s bash %s", domain, ip, patchScript)
	if err = utils.ExecCommand("/bin/bash", "-c", cmd); err != nil {
		return err
	}

	return nil
}
