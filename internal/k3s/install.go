package k3s

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/net"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Install Deploy the K3s airgap
func Install(spec *spec.K3sSpec) error {
	if err := setDefaultRoute(); err != nil {
		return err
	}

	if err := binaryInstall(spec); err != nil {
		return err
	}

	if err := helmInstall(spec); err != nil {
		return err
	}

	if err := imageImport(spec); err != nil {
		return err
	}

	installCmd := spec.K3sInstallCmd()

	log.Debugf("K3S install command: %s", installCmd)

	if err := utils.ExecCommand("/bin/bash", "-c", installCmd); err != nil {
		return err
	}

	if err := utils.CopyFile("/etc/rancher/k3s/k3s.yaml", "/root/.kube/config"); err != nil {
		return err
	}

	newDataDir := resolveK3sDataDir(spec)
	if newDataDir != "/var/lib/rancher/k3s" {
		// Ensure crictl can always find config at /etc/crictl.yaml
		target := "/etc/crictl.yaml"
		src := fmt.Sprintf("%s/agent/etc/crictl.yaml", newDataDir)

		// Create symlink only if source exists (k3s may not have started yet)
		if _, err := os.Stat(src); err == nil {
			// If target exists, check if it's already the correct symlink
			if fi, err := os.Lstat(target); err == nil {
				if fi.Mode()&os.ModeSymlink != 0 {
					// existing symlink
					existing, _ := os.Readlink(target)
					if existing == src {
						// already correct, nothing to do
						return nil
					}
				}
				// remove incorrect or regular file
				_ = os.Remove(target)
			}

			if err := os.Symlink(src, target); err != nil {
				return fmt.Errorf("failed to create crictl symlink: %w", err)
			}
		} else {
			log.Warnf("crictl config not found at %s, skip symlink", src)
		}
	}

	return nil
}

func setDefaultRoute() error {
	cmdStr := `
ip link show dummy0 >/dev/null 2>&1 || (ip link add dummy0 type dummy &&
ip link set dummy0 up &&
ip addr add 203.0.113.254/31 dev dummy0 &&
ip route add default via 203.0.113.255 dev dummy0 metric 1000)
`

	_ = utils.ExecCommand("bash", "-c", cmdStr)
	return nil
}

func imageImport(spec *spec.K3sSpec) error {
	defaultK3sDataDir := resolveK3sDataDir(spec)
	defaultK3sImagesDir := filepath.Join(defaultK3sDataDir, "agent/images/")

	if err := os.MkdirAll(defaultK3sImagesDir, 0755); err != nil {
		return err
	}

	srcImageFilePath := spec.K3sImageFilePath()
	dstImageFilePath := filepath.Join(defaultK3sImagesDir, spec.K3sImageFileName())

	if _, err := os.Stat(srcImageFilePath); os.IsNotExist(err) {
		return nil
	}

	if err := utils.CopyFile(srcImageFilePath, dstImageFilePath); err != nil {
		return err
	}

	// Create an empty .cache.json file
	cacheFile := filepath.Join(defaultK3sImagesDir, ".cache.json")
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		_ = os.WriteFile(cacheFile, []byte{}, 0644)
	}

	return nil
}

func binaryInstall(spec *spec.K3sSpec) error {
	defaultK3sBinDir := "/usr/local/bin/"
	if err := os.MkdirAll(defaultK3sBinDir, 0755); err != nil {
		return err
	}

	srcBinFilePath := spec.K3sBinFilePath()
	dstImageFilePath := filepath.Join(defaultK3sBinDir, filepath.Base(spec.K3sBinFilePath()))

	if _, err := os.Stat(srcBinFilePath); os.IsNotExist(err) {
		return err
	}

	if err := utils.CopyFile(srcBinFilePath, dstImageFilePath); err != nil {
		return err
	}

	return nil
}

func helmInstall(spec *spec.K3sSpec) error {
	defaultK3sBinDir := "/usr/local/bin/"
	if err := os.MkdirAll(defaultK3sBinDir, 0755); err != nil {
		return err
	}

	srcHelmFilePath := spec.K3sHelmFilePath()
	if err := utils.ExtractTarGz(srcHelmFilePath, defaultK3sBinDir); err != nil {
		return err
	}

	srcHelmBinFile := filepath.Join(defaultK3sBinDir, fmt.Sprintf("%s-%s", spec.OS, spec.Arch), "helm")
	destHelmBinFile := filepath.Join(defaultK3sBinDir, "helm")

	if _, err := os.Stat(srcHelmBinFile); os.IsNotExist(err) {
		return err
	}

	if err := utils.CopyFile(srcHelmBinFile, destHelmBinFile); err != nil {
		return err
	}

	_ = os.RemoveAll(filepath.Dir(srcHelmBinFile))

	return nil
}

func PatchInstall(spec *spec.K3sSpec, domain string) error {
	patchInstallScriptPath := filepath.Join(spec.K3sScriptDir(), "k3s-coredns-patch.sh")

	defaultIfaceName, err := net.GetDefaultInterface()
	if err != nil {
		return err
	}

	ip, err := net.GetInterfaceIP(defaultIfaceName)
	if err != nil {
		return err
	}

	cmd := fmt.Sprintf("DOMAIN=%s TARGET_IP=%s bash %s", domain, ip, patchInstallScriptPath)
	if err = utils.ExecCommand("/bin/bash", "-c", cmd); err != nil {
		return err
	}

	return nil
}

func resolveK3sDataDir(spec *spec.K3sSpec) string {
	dataDir := "/var/lib/rancher/k3s"

	for i, arg := range spec.InstallArgs {
		// Handle "--data-dir=/path"
		if strings.HasPrefix(arg, "--data-dir=") {
			v := strings.TrimPrefix(arg, "--data-dir=")
			if v != "" {
				dataDir = v
			}
			break
		}

		// Handle "--data-dir /path"
		if arg == "--data-dir" && i+1 < len(spec.InstallArgs) {
			v := spec.InstallArgs[i+1]
			if v != "" {
				dataDir = v
			}
			break
		}
	}

	return strings.TrimRight(dataDir, "/")
}
