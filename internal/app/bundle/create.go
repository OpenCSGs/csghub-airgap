package bundle

import (
	"csghub-airgap/internal/app/metadata"
	"csghub-airgap/internal/chart"
	"csghub-airgap/internal/config"
	iu "csghub-airgap/internal/image/utils"
	"csghub-airgap/internal/infra/os"
	"csghub-airgap/internal/infra/pkg"
	"csghub-airgap/internal/k3s"
	"csghub-airgap/internal/log"
	"runtime"
	"time"
)

func (s *Spec) Create() error {
	start := time.Now()

	if err := os.CheckArch(); err != nil {
		return err
	}

	if s.Arch == runtime.GOARCH {
		if err := pkg.Download(config.DefaultBaseDir); err != nil {
			return err
		}
	} else {
		log.Warnf("Skipping package download: cross-architecture download is not supported (target=%s, host=%s)", s.Arch, runtime.GOARCH)
		log.Warnf("Please download the required packages for the target architecture manually: https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html")
	}

	if !s.SkipK3s {
		if err := k3s.Download(s.K3s); err != nil {
			return err
		}
	}

	for _, c := range s.Charts {
		if !c.Enabled {
			log.Debugf("Skipping disabled chart %s", c.ChartName)
			continue
		}
		if c.ChartVersion == "" {
			c.ChartVersion = chart.LatestVersion(&c)
		}

		log.Debugf("Processing chart %s with version %s", c.ChartName, c.ChartVersion)

		if err := chart.Pull(&c); err != nil {
			return err
		}

		var opt iu.ExtractOption
		switch c.RepoName {
		case "csghub":
			opt = chart.CsghubExtractOption
		case "longhorn":
			opt = chart.LonghornExtractOption
		default:
		}

		var runtimeImages []string
		if c.RepoName == "csghub" {
			runtimeImages = s.SelectedRuntimeImages
		}
		if err := chart.Image(&c, runtimeImages, opt); err != nil {
			return err
		}
	}

	if err := metadata.Save(s); err != nil {
		return err
	}

	log.Infof("Create finished in %s", time.Since(start))
	return nil
}
