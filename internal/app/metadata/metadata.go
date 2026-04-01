package metadata

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

func Save(v any) error {
	if err := os.MkdirAll(config.DefaultBaseDir, 0755); err != nil {
		return err
	}

	manifestFile := filepath.Join(config.DefaultBaseDir, "metadata.yaml")

	data, err := yaml.Marshal(v)
	if err != nil {
		return err
	}

	if err = os.WriteFile(manifestFile, data, 0644); err != nil {
		return err
	}

	log.Infof("Manifest created: %s", manifestFile)

	return nil
}

func Load() (*spec.BundleSpec, error) {
	metadataFile := filepath.Join(config.DefaultBaseDir, "metadata.yaml")
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return nil, err
	}

	var metadata spec.BundleSpec
	if err = yaml.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}
