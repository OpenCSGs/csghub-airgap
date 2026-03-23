package chart

import (
	"fmt"

	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart/loader"
)

func ShowValuesFromTgz(tgzPath string) ([]byte, error) {
	chart, err := loader.Load(tgzPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from tgz: %w", err)
	}

	values, err := yaml.Marshal(chart.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to convert chart values to YAML: %w", err)
	}

	return values, nil
}
