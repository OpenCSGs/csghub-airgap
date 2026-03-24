package bundle

import "csghub-airgap/internal/utils"

func Cleanup() error {
	return utils.ExecCommand("k3s-uninstall.sh")
}
