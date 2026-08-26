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
		switch {
		case s.SkipPkg:
			log.Infof("Skipping NVIDIA toolkit package download (--skip package)")
		case !nvdpEnabled(s.Charts):
			log.Infof("Skipping NVIDIA toolkit package download (nvdp chart disabled)")
		default:
			if err := pkg.Download(config.DefaultBaseDir); err != nil {
				return err
			}
		}
	} else {
		log.Warnf("Skipping package download: cross-architecture download is not supported (target=%s, host=%s)", s.Arch, runtime.GOARCH)
		log.Warnf("Download the required packages manually, e.g. apt/dnf/zypper install %s", pkg.PackageList())
		log.Warnf("Debian/Ubuntu repo: %s", pkg.DebianRepoURL)
		log.Warnf("RHEL/Fedora/openSUSE repo: %s", pkg.RpmRepoURL)
		log.Warnf("See install guide: %s", pkg.InstallGuideURL)
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

		// Pull the chart. Skipping a chart is expressed via --skip <name>
		// which flips c.Enabled=false (handled above), so this chart always
		// resolves its latest version and pulls.
		if c.ChartVersion == "" {
			c.ChartVersion = chart.LatestVersion(&c)
		}
		if err := chart.Pull(&c); err != nil {
			return err
		}

		log.Debugf("Processing chart %s with version %s", c.ChartName, c.ChartVersion)

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
		if err := chart.Image(&c, runtimeImages, opt, s.SkipImages, s.SkipRuntimeImages); err != nil {
			return err
		}
	}

	if err := metadata.Save(s); err != nil {
		return err
	}

	log.Infof("Create finished in %s", time.Since(start))
	return nil
}
