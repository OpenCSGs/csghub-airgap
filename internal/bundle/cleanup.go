package bundle

import "csghub-airgap/internal/utils"

func Cleanup() error {
	return utils.ExecCommand("/usr/local/bin/k3s-uninstall.sh")
}
