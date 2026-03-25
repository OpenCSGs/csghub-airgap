package spec

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/net"
	"csghub-airgap/internal/utils"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/mcuadros/go-defaults"
)

type K3sSpec struct {
	ReleaseURL   string   `yaml:"release_url" default:"https://ghfast.top/https://github.com/k3s-io/k3s/releases"`
	ScriptURL    string   `yaml:"script_url" default:"https://get.k3s.io"`
	Version      string   `yaml:"version" default:"v1.34.5+k3s1"`
	Arch         string   `yaml:"arch"`
	HelmURL      string   `yaml:"helm_url" default:"https://get.helm.sh"`
	HelmVersion  string   `yaml:"helm_version" default:"v3.20.1"`
	OS           string   `yaml:"os"`
	SupportArchs string   `yaml:"support_archs" default:"amd64,arm64"`
	DestDir      string   `yaml:"dest_dir"`
	InstallEnvs  []string `yaml:"install_envs"`
	InstallArgs  []string `yaml:"install_args"`
}

func NewK3sSpec(destDir string) *K3sSpec {
	defaultInterface, err := net.GetDefaultInterface()
	if err != nil {
		log.Fatal(err)
	}

	ks := &K3sSpec{
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
		DestDir: utils.NVL(destDir, config.DefaultDestDir),
		InstallEnvs: []string{
			"INSTALL_K3S_SKIP_DOWNLOAD=true",
		},
		InstallArgs: []string{
			"--node-name=k3s-master",
			"--disable=traefik",
			"--kubelet-arg=eviction-hard=",
			fmt.Sprintf("--flannel-iface=%s", defaultInterface),
		},
	}

	// Populate default values from struct tags
	defaults.SetDefaults(ks)

	log.Debugf("Initialized K3s spec: %s", func() []byte { b, _ := json.Marshal(ks); return b }())

	return ks
}

// Validate Checks the validity of all K3s configuration parameters
func (ks *K3sSpec) Validate() error {
	VersionRegex := regexp.MustCompile(`^v\d+.\d+.\d+\+k3s\d$`)
	if strings.TrimSpace(ks.Version) == "" || !VersionRegex.MatchString(ks.Version) {
		return fmt.Errorf("invalid k3s version: %s", ks.Version)
	}

	if !strings.Contains(ks.SupportArchs, ks.Arch) {
		return fmt.Errorf("invalid k3s support arch: %s", ks.SupportArchs)
	}

	return nil
}

func (ks *K3sSpec) K3sDownLoadURL() string {
	downloadURL, err := url.JoinPath(ks.ReleaseURL, "download", ks.Version)
	if err != nil {
		log.Fatalf("failed to generate k3s download URL %s", err)
	}

	return downloadURL
}

// K3sDir return K3s Offline path
func (ks *K3sSpec) K3sDir() string {
	return filepath.Join(ks.DestDir, "k3s")
}

func (ks *K3sSpec) K3sBinDir() string {
	return filepath.Join(ks.K3sDir(), "bin")
}

func (ks *K3sSpec) K3sBinFilePath() string {
	return filepath.Join(ks.K3sBinDir(), "k3s")
}

func (ks *K3sSpec) K3sHelmFilePath() string {
	helmFileName := fmt.Sprintf("helm-%s-%s-%s.tar.gz", ks.HelmVersion, ks.OS, ks.Arch)
	return filepath.Join(ks.K3sBinDir(), helmFileName)
}

func (ks *K3sSpec) K3sImageDir() string {
	return filepath.Join(ks.K3sDir(), "images")
}

func (ks *K3sSpec) K3sImageFileName() string {
	return fmt.Sprintf("k3s-airgap-images-%s.tar.zst", ks.Arch)
}

func (ks *K3sSpec) K3sImageFilePath() string {
	return filepath.Join(ks.K3sImageDir(), ks.K3sImageFileName())
}

func (ks *K3sSpec) K3sScriptDir() string {
	return filepath.Join(ks.K3sDir(), "scripts")
}

func (ks *K3sSpec) K3sInstallCmd() string {
	script := filepath.Join(ks.K3sScriptDir(), "k3s-install.sh")

	var envStr string
	for _, env := range ks.InstallEnvs {
		envStr += env + " "
	}

	var argStr string
	for _, arg := range ks.InstallArgs {
		argStr += arg + " "
	}

	return fmt.Sprintf("%s %s %s", envStr, script, argStr)
}
