package spec

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/infra/net"
	"csghub-airgap/internal/log"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
)

type K3sSpec struct {
	ReleaseURL  string   `yaml:"release_url" default:"https://ghfast.top/https://github.com/k3s-io/k3s/releases"`
	ScriptURL   string   `yaml:"script_url" default:"https://get.k3s.io"`
	HelmURL     string   `yaml:"helm_url" default:"https://get.helm.sh"`
	HelmVersion string   `yaml:"helm_version" default:"v3.20.1"`
	InstallEnvs []string `yaml:"install_envs"`
	InstallArgs []string `yaml:"install_args"`
	BaseDir     string   `yaml:"base_dir"`
}

func NewK3sSpec() *K3sSpec {
	defaultInterface, err := net.GetDefaultInterface()
	if err != nil {
		log.Fatalf("failed to get default interface: %v", err)
	}

	return &K3sSpec{
		ReleaseURL: "https://ghfast.top/https://github.com/k3s-io/k3s/releases",
		ScriptURL:  "https://get.k3s.io",
		InstallEnvs: []string{
			"INSTALL_K3S_VERSION=v1.34.5+k3s1",
		},
		InstallArgs: []string{
			"--node-name=k3s-master",
			"--disable=traefik",
			"--kubelet-arg=eviction-hard=",
			"--flannel-iface=" + defaultInterface,
		},
		HelmURL:     "https://get.helm.sh",
		HelmVersion: "v3.20.1",
		BaseDir:     config.DefaultBaseDir,
	}
}

func (ks *K3sSpec) Dir() string {
	return filepath.Join(ks.BaseDir, "k3s")
}

func (ks *K3sSpec) BaseURL() string {
	version, _ := ks.GetEnv("INSTALL_K3S_VERSION")

	baseURL, err := url.JoinPath(ks.ReleaseURL, "download", version)
	if err != nil {
		log.Fatalf("failed to generate k3s download URL %s", err)
	}

	return baseURL
}

func (ks *K3sSpec) BinaryName() string {
	return "k3s"
}

func (ks *K3sSpec) BinaryURL() string {
	binaryURL, err := url.JoinPath(ks.BaseURL(), ks.BinaryName())
	if err != nil {
		log.Fatalf("failed to generate k3s binary URL %s", err)
	}

	return binaryURL
}

func (ks *K3sSpec) BinaryPath() string {
	return filepath.Join(ks.Dir(), "bin", ks.BinaryName())
}

func (ks *K3sSpec) ImageName() string {
	return fmt.Sprintf("k3s-airgap-images-%s.tar.zst", runtime.GOARCH)
}

func (ks *K3sSpec) ImageURL() string {
	imageURL, err := url.JoinPath(ks.BaseURL(), ks.ImageName())
	if err != nil {
		log.Fatalf("failed to generate k3s image URL %s", err)
	}

	return imageURL
}

func (ks *K3sSpec) ImagePath() string {
	return filepath.Join(ks.Dir(), "images", ks.ImageName())
}

func (ks *K3sSpec) ScriptPath() string {
	return filepath.Join(ks.Dir(), "scripts", "k3s-install.sh")
}

func (ks *K3sSpec) HelmBinaryName() string {
	return fmt.Sprintf("helm-%s-%s-%s.tar.gz", ks.HelmVersion, runtime.GOOS, runtime.GOARCH)
}

func (ks *K3sSpec) HelmBinaryURL() string {
	helmURL, err := url.JoinPath(ks.HelmURL, ks.HelmBinaryName())
	if err != nil {
		log.Fatalf("failed to generate helm URL %s", err)
	}

	return helmURL
}

func (ks *K3sSpec) HelmBinaryPath() string {
	return filepath.Join(ks.Dir(), "helm", ks.HelmBinaryName())
}

func (s *K3sSpec) GetEnv(key string) (string, bool) {
	for _, env := range s.InstallEnvs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 && parts[0] == key {
			return parts[1], true
		}
	}
	return "", false
}

func (s *K3sSpec) GetArg(key string) (string, bool) {
	for _, env := range s.InstallArgs {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 && parts[0] == key {
			return parts[1], true
		}
	}
	return "", false
}

func (s *K3sSpec) InstallCmd() string {
	return strings.TrimSpace(
		fmt.Sprintf("%s %s %s",
			strings.Join(s.InstallEnvs, " "),
			s.ScriptPath(),
			strings.Join(s.InstallArgs, " "),
		),
	)
}
