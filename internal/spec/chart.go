package spec

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/manifest"
	"csghub-airgap/internal/utils"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type ChartSpec struct {
	Name         string                 `yaml:"name"`
	Enabled      bool                   `yaml:"enabled"`
	RepoName     string                 `yaml:"repo_name"`     // Helm repo name
	RepoURL      string                 `yaml:"repo_url"`      // Helm repo URL
	ChartName    string                 `yaml:"chart_name"`    // Chart name in the repo
	ChartVersion string                 `yaml:"chart_version"` // Version, can be empty for latest
	Values       map[string]interface{} `yaml:"values"`
	Weight       int                    `yaml:"weight,omitempty"`
	DependsOn    []string               `yaml:"depends_on,omitempty"`
	// ExtraImages lists image refs to bundle even though the chart does not
	// declare them (e.g. runtime-injected sidecars like argo's argoexec
	// image, which is referenced by workflow-controller at workflow pod
	// creation time but never appears as a chart-rendered `image:` field).
	// These go through the same registry-rewrite and cache paths as images
	// extracted from the chart.
	ExtraImages []string `yaml:"extra_images,omitempty"`
}

// NewChartSpec takes a slice of YAML strings representing one or more chart manifests
// and parses them into a slice of ChartSpec structures.
//
// Each element in the 'manifests' slice should be the full YAML content of a chart.
// The YAML content is expected to represent a single chart manifest (mapping) with fields
// like 'name', 'enabled', 'repo_name', 'repo_url', 'chart_name', 'chart_version',
// 'values', 'weight', and 'dependsOn'.
//
// The function returns a slice of ChartSpec, where each parsed chart manifest becomes
// one ChartSpec in the slice. Empty or whitespace-only strings in the slice are ignored.
func NewChartSpec() ([]ChartSpec, error) {
	manifests, err := manifest.LoadYAML()
	if err != nil {
		return nil, err
	}

	var allCharts []ChartSpec

	for _, mft := range manifests {
		if strings.TrimSpace(mft) == "" {
			continue
		}

		var charts ChartSpec
		if err := yaml.Unmarshal([]byte(mft), &charts); err != nil {
			return nil, fmt.Errorf("failed to parse YAML %s: %v", mft, err)
		}

		allCharts = append(allCharts, charts)
	}

	return allCharts, nil
}

func (cs *ChartSpec) ChartRef() string {
	if cs.RepoName != "" && cs.ChartName != "" {
		return cs.RepoName + "/" + cs.ChartName
	}

	log.Fatalf("Invalid RepoName or ChartName")
	return ""
}

func (cs *ChartSpec) ChartDir() string {
	return filepath.Join(config.DefaultBaseDir, cs.ChartName, "charts")
}

func (cs *ChartSpec) ChartFileName() string {
	return fmt.Sprintf("%s-%s.tgz", cs.ChartName, cs.ChartVersion)
}

func (cs *ChartSpec) ChartFilePath() string {
	return filepath.Join(cs.ChartDir(), cs.ChartFileName())
}

func (cs *ChartSpec) Merge(ds *ChartSpec) {
	if ds == nil {
		return
	}

	av := reflect.ValueOf(cs).Elem()
	bv := reflect.ValueOf(ds).Elem()

	for i := 0; i < av.NumField(); i++ {
		fieldA := av.Field(i)
		fieldB := bv.Field(i)

		if !fieldB.IsValid() || utils.IsZeroValue(fieldB) || !fieldA.CanSet() {
			continue
		}

		// 👇 Special ChartOptions
		if av.Type().Field(i).Name == "ChartOptions" {
			if fieldA.IsNil() {
				fieldA.Set(fieldB)
			} else {
				merged := utils.MergeMap(
					fieldA.Interface().(map[string]interface{}),
					fieldB.Interface().(map[string]interface{}),
				)
				fieldA.Set(reflect.ValueOf(merged))
			}
			continue
		}

		fieldA.Set(fieldB)
	}
}
