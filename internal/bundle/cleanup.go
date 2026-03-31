package bundle

import "csghub-airgap/internal/infra/shell"

func Cleanup() error {
	return shell.Execute("/usr/local/bin/k3s-uninstall.sh")
}
