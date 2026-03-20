package bundle

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

func LoadFromManifest(destDir string, bs *spec.BundleSpec) (*Spec, error) {
	manifestFilePath := filepath.Join(destDir, "manifest.yaml")
	bundleSpec, err := loadSpecFromManifest(manifestFilePath)
	if err != nil {
		return nil, err
	}

	if bs.K3s != nil {
		bundleSpec.K3s = bs.K3s
	}

	if bs.Csghub != nil {
		bundleSpec.Csghub = bs.Csghub
	}

	if bs.Longhorn != nil {
		bundleSpec.Longhorn = bs.Longhorn
	}

	if bs.Nvdp != nil {
		bundleSpec.Nvdp = bs.Nvdp
	}

	if bs.DestDir != "" {
		bundleSpec.DestDir = bs.DestDir
	}

	return bundleSpec, nil
}

func (s *Spec) Manifest() error {
	outDir := utils.NVL(s.DestDir, config.DefaultDestDir)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	manifestFile := filepath.Join(outDir, "manifest.yaml")

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	if err = os.WriteFile(manifestFile, data, 0644); err != nil {
		return err
	}

	log.Infof("Manifest created: %s", manifestFile)

	return nil
}

// loadSpecFromManifest reads the manifest.yaml file and parses it into a Spec.
// If srcDir is empty, it will use the CLI-specified directory or the default directory.
func loadSpecFromManifest(manifest string) (*Spec, error) {
	// Read the manifest.yaml file
	data, err := os.ReadFile(manifest)
	if err != nil {
		return nil, err
	}

	var s Spec
	// Unmarshal YAML content into Spec
	if err = yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return &s, nil
}
