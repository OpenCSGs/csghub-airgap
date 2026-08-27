package k3s

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/infra/net"
	"csghub-airgap/internal/infra/shell"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"csghub-airgap/internal/utils/extract"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func Install(ks *spec.K3sSpec) error {
	defaultDataDir, _ := ks.GetArg("--data-dir")
	dataDir := utils.NVL(defaultDataDir, "/var/lib/rancher/k3s")

	if config.DryRun {
		helmOS := ks.TargetOS
		if helmOS == "" {
			helmOS = runtime.GOOS
		}
		log.Infof("[DRY-RUN] $ install -m 0755 %s /usr/local/bin/k3s", ks.BinaryPath())
		log.Infof("[DRY-RUN] $ install -m 0644 %s %s", ks.ImagePath(), filepath.Join(dataDir, "agent/images"))
		log.Infof("[DRY-RUN] $ tar -xzf %s -C /usr/local/bin", ks.HelmBinaryPath())
		log.Infof("[DRY-RUN] $ mv /usr/local/bin/%s-%s/helm /usr/local/bin/helm && rm -rf /usr/local/bin/%s-%s", helmOS, runtime.GOARCH, helmOS, runtime.GOARCH)
		log.Infof("[DRY-RUN] $ %s", ks.InstallCmd())
		log.Infof("[DRY-RUN] $ cp /etc/rancher/k3s/k3s.yaml /root/.kube/config")
		log.Infof("[DRY-RUN] $ ln -sf %s /etc/crictl.yaml", filepath.Join(dataDir, "agent/etc/crictl.yaml"))
		return nil
	}

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

			if err := extract.Untar(asset.Src, asset.Dest); err != nil {
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
	if err := net.SetDefaultRoute(); err != nil {
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

	if err := shell.Execute("/bin/bash", "-c", ks.InstallCmd()); err != nil {
		return err
	}

	// 6. Set default k8s context
	if err := utils.CopyFile("/etc/rancher/k3s/k3s.yaml", "/root/.kube/config"); err != nil {
		return err
	}

	// 7. local CoreDNS resolve patch
	// Always resolve crictl config from dataDir (which falls back to the k3s
	// default), so the symlink is created even when --data-dir is not set.
	{
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
