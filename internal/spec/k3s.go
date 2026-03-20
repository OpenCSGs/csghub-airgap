package spec

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/utils"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/mcuadros/go-defaults"
)

type K3sSpec struct {
	ReleaseURL   string `yaml:"release_url" default:"https://github.com/k3s-io/k3s/releases"`
	ScriptURL    string `yaml:"script_url" default:"https://get.k3s.io"`
	Version      string `yaml:"version" default:"v1.34.5+k3s1"`
	Arch         string `yaml:"arch"`
	HelmURL      string `yaml:"helm_url" default:"https://get.helm.sh"`
	HelmVersion  string `yaml:"helm_version" default:"v3.20.1"`
	OS           string `yaml:"os"`
	SupportArchs string `yaml:"support_archs" default:"amd64,arm64"`
	DestDir      string `yaml:"dest_dir"`
}

func NewK3sSpec(destDir string) *K3sSpec {
	ks := &K3sSpec{
		Arch:    runtime.GOARCH,
		OS:      runtime.GOOS,
		DestDir: utils.NVL(destDir, config.DefaultDestDir),
	}

	// Populate default values from struct tags
	defaults.SetDefaults(ks)

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

// K3sDir return K3s Offline path
func (ks *K3sSpec) K3sDir() string {
	return filepath.Join(ks.DestDir, "k3s")
}
