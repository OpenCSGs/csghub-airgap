package bundle

import (
	"csghub-airgap/internal/chart"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/k3s"
	"csghub-airgap/internal/spec"
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

		if err := chart.Pull(c); err != nil {
			return err
		}

		var opt image.ExtractOption
		if c.RepoName == "longhorn" {
			opt = chart.LonghornExtractOption
		} else {
			opt = image.ExtractOption{}
		}

		if err := chart.Image(c, opt); err != nil {
			return err
		}
	}

	if err := s.Manifest(); err != nil {
		return err
	}

	return nil
}
