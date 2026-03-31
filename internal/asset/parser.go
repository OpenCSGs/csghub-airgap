package asset

import (
	"csghub-airgap/internal/spec"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser takes a slice of YAML strings representing one or more chart manifests
// and parses them into a slice of ChartSpec structures.
//
// Each element in the 'manifests' slice should be the full YAML content of a chart.
// The YAML content is expected to represent a single chart manifest (mapping) with fields
// like 'name', 'enabled', 'repo_name', 'repo_url', 'chart_name', 'chart_version',
// 'values', 'weight', and 'dependsOn'.
//
// The function returns a slice of ChartSpec, where each parsed chart manifest becomes
// one ChartSpec in the slice. Empty or whitespace-only strings in the slice are ignored.
func Parser(manifests []string) ([]spec.ChartSpec, error) {
	var allCharts []spec.ChartSpec

	for _, manifest := range manifests {
		if strings.TrimSpace(manifest) == "" {
			continue
		}

		var charts spec.ChartSpec
		if err := yaml.Unmarshal([]byte(manifest), &charts); err != nil {
			return nil, fmt.Errorf("failed to parse YAML %s: %w", manifest, err)
		}

		allCharts = append(allCharts, charts)
	}

	return allCharts, nil
}
