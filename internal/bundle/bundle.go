package bundle

import (
	"csghub-airgap/internal/chart"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/k3s"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"
)

type Spec spec.BundleSpec

func NewFromSpec(bs *spec.BundleSpec) *Spec {
	return &Spec{
		K3s:          bs.K3s,
		SkipK3s:      bs.SkipK3s,
		Csghub:       bs.Csghub,
		Longhorn:     bs.Longhorn,
		SkipLonghorn: bs.SkipLonghorn,
		Nvdp:         bs.Nvdp,
		SkipNvdp:     bs.SkipNvdp,
		Arch:         bs.Arch,
		DestDir:      bs.DestDir,
	}
}

func (s *Spec) Create() error {
	if err := checkArch(); err != nil {
		return err
	}

	if !s.SkipK3s {
		if err := k3s.DownloadK3S(s.K3s); err != nil {
			return err
		}
	}

	var charts []*spec.ChartSpec

	if !s.SkipLonghorn {
		charts = append(charts, s.Longhorn)
	}

	if !s.SkipNvdp {
		charts = append(charts, s.Nvdp)
	}

	charts = append(charts, s.Csghub)

	for _, c := range charts {
		if c.ChartVersion == "" {
			c.ChartVersion = chart.LastVersion(c).String()
		}

		if err := chart.Download(c); err != nil {
			return err
		}

		manifest, err := chart.Template(c)
		if err != nil {
			return err
		}

		images := image.Extract(manifest)

		imageSpec := &spec.ImageSpec{
			Images:       images,
			OutPath:      utils.NVL(s.DestDir, config.DefaultDestDir),
			ChartName:    c.ChartName,
			ChartVersion: c.ChartVersion,
		}
		if err = image.Export(imageSpec); err != nil {
			return err
		}
	}

	if err := s.Manifest(); err != nil {
		return err
	}

	return nil
}

func (s *Spec) Deploy() error {
	if err := checkArch(); err != nil {
		return err
	}

	if !s.SkipK3s {
		if err := k3s.Install(s.K3s); err != nil {
			return err
		}
	}

	return nil
}

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
