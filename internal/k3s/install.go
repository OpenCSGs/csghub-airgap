package k3s

import (
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"fmt"
	"os"
	"path/filepath"
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

	if err := utils.ExecCommand("/bin/bash", "-c", installCmd); err != nil {
		return err
	}

	if err := utils.CopyFile("/etc/rancher/k3s/k3s.yaml", "/root/.kube/config"); err != nil {
		return err
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
	defaultK3sImagesDir := "/var/lib/rancher/k3s/agent/images/"
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
