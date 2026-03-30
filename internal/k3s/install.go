package k3s

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func Install(ks *spec.K3sSpec) error {
	defaultDataDir, exists := ks.GetArg("--data-dir")
	dataDir := utils.NVL(defaultDataDir, "/var/lib/rancher/k3s")

	assets := []Asset{
		{
			Src:        ks.BinaryPath(),
			Dest:       "/usr/local/bin",
			Executable: true,
		},
		{
			Src:  ks.ImagePath(),
			Dest: filepath.Join(dataDir, "agent/images"),
		},
		{
			Src:        ks.HelmBinaryPath(),
			Dest:       "/usr/local/bin",
			Executable: true,
		},
	}

	for _, asset := range assets {
		dstFile := filepath.Join(asset.Dest, filepath.Base(asset.Src))

		if err := os.MkdirAll(asset.Dest, 0755); err != nil {
			return err
		}

		if _, err := os.Stat(asset.Src); err != nil {
			return fmt.Errorf("source file %s not found", asset.Src)
		}

		src := asset.Src

		if asset.Src == ks.HelmBinaryPath() {
			dstFile = filepath.Join(asset.Dest, "helm")

			if _, err := os.Stat(dstFile); err == nil {
				continue
			}

			if err := utils.ExtractTarGz(asset.Src, asset.Dest); err != nil {
				return err
			}

			src = filepath.Join(asset.Dest, fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH), "helm")
		}

		if _, err := os.Stat(dstFile); err == nil {
			continue
		}

		if err := utils.CopyFile(src, dstFile); err != nil {
			return err
		}

		if asset.Executable {
			if err := os.Chmod(dstFile, 0755); err != nil {
				return err
			}
		}

		// cleanup helm extract dir
		if asset.Src == ks.HelmBinaryPath() {
			_ = os.RemoveAll(filepath.Join(asset.Dest, fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)))
		}
	}

	imageCacheFile := filepath.Join(dataDir, "agent/images", ".cache.json")
	if _, err := os.Stat(imageCacheFile); os.IsNotExist(err) {
		if err := os.WriteFile(imageCacheFile, []byte{}, 0644); err != nil {
			return err
		}
	}

	// 1. Set default routes
	if err := setDefaultRoute(); err != nil {
		return err
	}

	// 2. K3S binary install
	// Already copied

	// 3. K3S images import
	// Already copied

	// 4. Helm install
	// Already copied

	// 5. Exec Install
	log.Debugf("K3S install command: %s", ks.InstallCmd())

	if err := utils.ExecCommand("/bin/bash", "-c", ks.InstallCmd()); err != nil {
		return err
	}

	// 6. Set default k8s context
	if err := utils.CopyFile("/etc/rancher/k3s/k3s.yaml", "/root/.kube/config"); err != nil {
		return err
	}

	// 7. local CoreDNS resolve patch

	if exists && defaultDataDir != "" {
		defaultCrictlConfig := filepath.Join(dataDir, "agent/etc/crictl.yaml")
		targetCrictlConfig := "/etc/crictl.yaml"

		if _, err := os.Stat(defaultCrictlConfig); os.IsNotExist(err) {
			log.Warnf("crictl config not found at %s, skip symlink", defaultCrictlConfig)
			return nil
		}

		_ = os.Remove(targetCrictlConfig)

		if err := os.Symlink(defaultCrictlConfig, targetCrictlConfig); err != nil {
			return err
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
