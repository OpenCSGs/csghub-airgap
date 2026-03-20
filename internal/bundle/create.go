package bundle

import (
	"csghub-airgap/internal/chart"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/k3s"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
)

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
