package spec

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/utils"
	"fmt"
	"path/filepath"
)

type ChartSpec struct {
	RepoName     string                 `yaml:"repo_name"`     // Helm repo name
	RepoURL      string                 `yaml:"repo_url"`      // Helm repo URL
	ChartName    string                 `yaml:"chart_name"`    // Chart name in the repo
	ChartVersion string                 `yaml:"chart_version"` // Version, can be empty for latest
	OutPath      string                 `yaml:"out_path"`      // Local directory to store the chart
	ChartOptions map[string]interface{} `yaml:"chart_options"` // Render or install options
}

func NewChartSpec(destDir string) *ChartSpec {
	return &ChartSpec{
		OutPath: utils.NVL(destDir, config.DefaultDestDir),
	}
}

func (cs *ChartSpec) ChartRef() string {
	if cs.RepoName != "" && cs.ChartName != "" {
		return cs.RepoName + "/" + cs.ChartName
	}

	log.Fatalf("Invalid RepoName or ChartName")
	return ""
}

func (cs *ChartSpec) ChartDir() string {
	return filepath.Join(cs.OutPath, cs.ChartName, "charts")
}

func (cs *ChartSpec) ChartFileName() string {
	return fmt.Sprintf("%s-%s.tgz", cs.ChartName, cs.ChartVersion)
}

func (cs *ChartSpec) ChartFilePath() string {
	return filepath.Join(cs.ChartDir(), cs.ChartFileName())
}

func WithCsghubOptions(opts map[string]interface{}) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.Csghub == nil {
			bs.Csghub = NewCsghubSpec(bs.DestDir, nil)
		}
		bs.Csghub.ChartOptions = utils.MergeMap(bs.Csghub.ChartOptions, opts)
	}
}

func WithLonghornOptions(opts map[string]interface{}) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.Longhorn == nil {
			bs.Longhorn = NewLonghornSpec(bs.DestDir, nil)
		}
		bs.Longhorn.ChartOptions = utils.MergeMap(bs.Longhorn.ChartOptions, opts)
	}
}

func WithNvdpOptions(opts map[string]interface{}) BundleSpecOption {
	return func(bs *BundleSpec) {
		if bs.Nvdp == nil {
			bs.Nvdp = NewNvdpSpec(bs.DestDir, nil)
		}
		bs.Nvdp.ChartOptions = utils.MergeMap(bs.Nvdp.ChartOptions, opts)
	}
}
