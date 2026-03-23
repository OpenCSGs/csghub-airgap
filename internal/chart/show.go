package chart

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func ShowChartValues(chartRef string) ([]byte, error) {
	chart, err := loader.Load(chartRef)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from tgz: %w", err)
	}

	values, err := yaml.Marshal(chart.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to convert chart values to YAML: %w", err)
	}

	return values, nil
}

func ShowChartAll(chartRef string) ([]byte, error) {
	chart, err := loader.Load(chartRef)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from tgz: %w", err)
	}

	values, err := yaml.Marshal(chart.Raw)
	if err != nil {
		return nil, fmt.Errorf("failed to convert chart values to YAML: %w", err)
	}

	return values, nil
}

func ShowKindOf(manifest, targetKind string) ([]string, error) {
	var results []string

	docs := strings.Split(manifest, "---")
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		var obj map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			continue
		}

		kind, ok := obj["kind"].(string)
		if !ok {
			continue
		}

		if kind == targetKind {
			results = append(results, doc)
		}
	}

	return results, nil
}
