package spec

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/utils"
	"fmt"
	"runtime"
)

type BundleSpec struct {
	K3s          *K3sSpec   `yaml:"k3s"`
	SkipK3s      bool       `yaml:"skip-k3s"`
	Csghub       *ChartSpec `yaml:"csghub"`
	Longhorn     *ChartSpec `yaml:"longhorn"`
	SkipLonghorn bool       `yaml:"skip-longhorn"`
	Nvdp         *ChartSpec `yaml:"nvdp"`
	SkipNvdp     bool       `yaml:"skip-nvdp"`
	Arch         string     `yaml:"arch"`
	DestDir      string     `yaml:"dest_dir"`
}

type BundleSpecOption func(*BundleSpec)

func NewBundleSpec(opts ...BundleSpecOption) (*BundleSpec, error) {
	bs := &BundleSpec{
		SkipK3s:      false,
		SkipLonghorn: false,
		Arch:         runtime.GOARCH,
		DestDir:      config.DefaultDestDir,
	}

	if bs.K3s == nil {
		bs.K3s = NewK3sSpec(bs.DestDir)
	}
	if bs.Csghub == nil {
		bs.Csghub = NewCsghubSpec(bs.DestDir, nil)
	}
	if bs.Longhorn == nil {
		bs.Longhorn = NewLonghornSpec(bs.DestDir, nil)
	}
	if bs.Nvdp == nil {
		bs.Nvdp = NewNvdpSpec(bs.DestDir, nil)
	}

	for _, o := range opts {
		o(bs)
	}

	return bs, nil
}

func NewCsghubSpec(destDir string, options map[string]interface{}) *ChartSpec {
	cs := &ChartSpec{
		RepoName:     "csghub",
		RepoURL:      "https://charts.opencsg.com/csghub/",
		ChartName:    "csghub",
		ChartVersion: "",
		OutPath:      utils.NVL(destDir, config.DefaultDestDir),
		ChartOptions: map[string]interface{}{
			"global": map[string]interface{}{
				"image": map[string]interface{}{
					"registry": config.DefaultRegistry,
				},
				"imageRegistry": config.DefaultRegistryWithNamespace,
				"edition":       "ee",
				"gateway": map[string]interface{}{
					"external": map[string]interface{}{
						"domain": "csghub.example.com",
					},
				},
			},
			"server": map[string]interface{}{
				"gitlabShell": map[string]interface{}{
					"sshPort": "2222",
				},
			},
			"runner": map[string]interface{}{
				"registry": map[string]interface{}{
					"repository": "csghub",
				},
				"volcano": map[string]interface{}{
					"basic": map[string]interface{}{
						"image_pull_policy": "IfNotPresent",
					},
				},
				"argo": map[string]interface{}{
					"images": map[string]interface{}{
						"pullPolicy": "IfNotPresent",
					},
					"controller": map[string]interface{}{
						"image": map[string]interface{}{
							"registry": config.DefaultRegistryWithNamespace,
						},
					},
					"executor": map[string]interface{}{
						"image": map[string]interface{}{
							"registry": config.DefaultRegistryWithNamespace,
						},
					},
					"server": map[string]interface{}{
						"image": map[string]interface{}{
							"registry": config.DefaultRegistryWithNamespace,
						},
					},
				},
			},
		},
	}

	if options != nil {
		cs.ChartOptions = utils.MergeMap(cs.ChartOptions, options)
	}

	return cs
}

func NewLonghornSpec(destDir string, options map[string]interface{}) *ChartSpec {
	return &ChartSpec{
		RepoName:     "longhorn",
		RepoURL:      "https://charts.opencsg.com/repository/longhorn/",
		ChartName:    "longhorn",
		ChartVersion: "",
		OutPath:      utils.NVL(destDir, config.DefaultDestDir),
		ChartOptions: utils.NVLMap(options, map[string]interface{}{
			"global": map[string]interface{}{
				"imageRegistry": config.DefaultRegistryWithNamespace,
			},
			"persistence": map[string]interface{}{
				"backupTargetName":         "",
				"defaultClassReplicaCount": 1,
				"defaultDataLocality":      "best-effort",
			},
			"defaultSettings": map[string]interface{}{
				"defaultDataPath":                         "/var/lib/longhorn/",
				"storageOverProvisioningPercentage":       3000, // Allow 30 times overuse
				"storageMinimalAvailablePercentage":       10,
				"storageReservedPercentageForDefaultDisk": 10,
			},
		}),
	}
}

func NewNvdpSpec(destDir string, options map[string]interface{}) *ChartSpec {
	return &ChartSpec{
		RepoName:     "nvdp",
		RepoURL:      "https://charts.opencsg.com/k8s-device-plugin/",
		ChartName:    "nvidia-device-plugin",
		ChartVersion: "",
		OutPath:      utils.NVL(destDir, config.DefaultDestDir),
		ChartOptions: utils.NVLMap(options, map[string]interface{}{
			"nfd": map[string]interface{}{
				"image": map[string]interface{}{
					"repository": fmt.Sprintf("%s/nfd/node-feature-discovery", config.DefaultRegistryWithNamespace),
				},
			},
			"image": map[string]interface{}{
				"repository": fmt.Sprintf("%s/nvidia/k8s-device-plugin", config.DefaultRegistryWithNamespace),
			},
			"gfd": map[string]interface{}{
				"enabled": true,
			},
			"runtimeClassName": "nvidia",
		}),
	}
}

func setDefaultChart(spec **ChartSpec, newFn func(string, map[string]interface{}) *ChartSpec, destDir string) {
	if *spec == nil {
		*spec = newFn(destDir, nil)
	}
}

func WithDestDir(dir string) BundleSpecOption {
	return func(bs *BundleSpec) {
		bs.DestDir = dir
		bs.K3s.DestDir = dir
		bs.Csghub.OutPath = dir
		bs.Longhorn.OutPath = dir
		bs.Nvdp.OutPath = dir
	}
}

func WithK3sVersion(ver string) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.K3s == nil {
			bs.K3s = NewK3sSpec(bs.DestDir)
		}
		bs.K3s.Version = ver
	}
}

func WithK3sSkip(skip bool) BundleSpecOption {
	return func(bs *BundleSpec) {
		bs.SkipK3s = skip
	}
}

func WithK3sEnvs(envs []string) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.K3s == nil {
			bs.K3s = NewK3sSpec(bs.DestDir)
		}
		bs.K3s.InstallEnvs = append(bs.K3s.InstallEnvs, envs...)
	}
}

func WithK3sArgs(args []string) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.K3s == nil {
			bs.K3s = NewK3sSpec(bs.DestDir)
		}

		if !bs.SkipNvdp {
			bs.K3s.InstallArgs = append(
				bs.K3s.InstallArgs,
				"--default-runtime=nvidia",
			)
		}

		bs.K3s.InstallArgs = append(bs.K3s.InstallArgs, args...)
	}
}

func WithCsghubVersion(ver string) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.Csghub == nil {
			bs.Csghub = NewCsghubSpec(bs.DestDir, nil)
		}
		bs.Csghub.ChartVersion = ver
	}
}

func WithLonghornVersion(ver string) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.Longhorn == nil {
			bs.Longhorn = NewLonghornSpec(bs.DestDir, nil)
		}
		bs.Longhorn.ChartVersion = ver
	}
}

func WithLonghornSkip(skip bool) BundleSpecOption {
	return func(bs *BundleSpec) {
		bs.SkipLonghorn = skip
	}
}

func WithNvdpVersion(ver string) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.Nvdp == nil {
			bs.Nvdp = NewNvdpSpec(bs.DestDir, nil)
		}
		bs.Nvdp.ChartVersion = ver
	}
}

func WithNvdpSkip(skip bool) BundleSpecOption {
	return func(bs *BundleSpec) {
		bs.SkipNvdp = skip
	}
}
