package spec

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/utils"
	"runtime"
)

type BundleSpec struct {
	K3s      *K3sSpec   `yaml:"k3s"`
	Csghub   *ChartSpec `yaml:"csghub"`
	Longhorn *ChartSpec `yaml:"longhorn"`
	Nvdp     *ChartSpec `yaml:"nvdp"`
	Arch     string     `yaml:"arch"`
	DestDir  string     `yaml:"dest_dir"`
}

type BundleSpecOption func(*BundleSpec)

func NewBundleSpec(opts ...BundleSpecOption) (*BundleSpec, error) {
	bs := &BundleSpec{
		Arch:    runtime.GOARCH,
		DestDir: config.DefaultDestDir,
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
					"registry": "opencsg-registry.cn-beijing.cr.aliyuncs.com",
				},
				"imageRegistry": "opencsg-registry.cn-beijing.cr.aliyuncs.com/opencsghq",
			},
			"runner": map[string]interface{}{
				"registry": map[string]interface{}{
					"repository": "csghub",
				},
				"lws": map[string]interface{}{
					"image": map[string]interface{}{
						"manager": map[string]interface{}{
							"repository": "opencsg-registry.cn-beijing.cr.aliyuncs.com/opencsghq/lws",
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
				"imageRegistry": "docker.1ms.run",
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
					"repository": "opencsg-registry.cn-beijing.cr.aliyuncs.com/opencsghq/nfd/node-feature-discovery",
				},
			},
			"image": map[string]interface{}{
				"repository": "opencsg-registry.cn-beijing.cr.aliyuncs.com/opencsghq/nvidia/k8s-device-plugin",
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

func WithNvdpVersion(ver string) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.Nvdp == nil {
			bs.Nvdp = NewNvdpSpec(bs.DestDir, nil)
		}
		bs.Nvdp.ChartVersion = ver
	}
}
