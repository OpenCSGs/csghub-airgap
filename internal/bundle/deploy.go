package bundle

import (
	"csghub-airgap/internal/chart"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/k3s"
	"csghub-airgap/internal/kernel"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/pack"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
)

func (s *Spec) Deploy() error {
	if err := checkArch(); err != nil {
		return err
	}

	if err := kernel.SetSysctl(nil); err != nil {
		return err
	}

	if !s.SkipK3s {
		if err := k3s.Install(s.K3s); err != nil {
			return err
		}
	}

	var charts []*spec.ChartSpec

	if !s.SkipLonghorn {
		charts = append(charts, s.Longhorn)
		if err := pack.InstallPackages(s.DestDir); err != nil {
			return err
		}
	}

	if !s.SkipNvdp {
		charts = append(charts, s.Nvdp)
		log.Warn("NVIDIA GPU are not included in the Airgap installation")
	}

	charts = append(charts, s.Csghub)

	for _, c := range charts {
		imageSpec := &spec.ImageSpec{
			Images:       nil,
			OutPath:      utils.NVL(s.DestDir, config.DefaultDestDir),
			ChartName:    c.ChartName,
			ChartVersion: c.ChartVersion,
		}
		if err := image.Import(imageSpec); err != nil {
			return err
		}

		if chart.List(c) {
			if err := chart.Upgrade(c); err != nil {
				return err
			}
		} else {
			if err := chart.Install(c); err != nil {
				return err
			}
		}
	}

	return nil
}
