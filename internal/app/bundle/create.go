package bundle

import (
	"csghub-airgap/internal/app/metadata"
	"csghub-airgap/internal/chart"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	iu "csghub-airgap/internal/image/utils"
	osi "csghub-airgap/internal/infra/os"
	"csghub-airgap/internal/infra/pkg"
	"csghub-airgap/internal/k3s"
	"csghub-airgap/internal/log"
	"os"
	"runtime"
	"time"
)

func (s *Spec) Create() error {
	start := time.Now()

	if err := osi.CheckArch(); err != nil {
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

		if s.SkipCharts {
			// Prefer reusing an already-pulled chart file; if it is missing
			// (first run with --skip charts, or image-only mode), pull it.
			if c.ChartVersion == "" {
				c.ChartVersion = chart.LatestVersion(&c)
			}
			if _, err := os.Stat(c.ChartFilePath()); err == nil {
				log.Debugf("Skipping chart pull for %s (--skip charts, chart exists)", c.ChartName)
			} else {
				log.Infof("Chart %s not found; pulling despite --skip charts", c.ChartName)
				if err := chart.Pull(&c); err != nil {
					return err
				}
			}
		} else {
			if c.ChartVersion == "" {
				c.ChartVersion = chart.LatestVersion(&c)
			}
			if err := chart.Pull(&c); err != nil {
				return err
			}
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

		// Image-only mode: --skip k3s,package,charts and image not skipped.
		// Prompt the user with a TUI to pick which main images to export.
		var selectedMain []string
		if s.SkipCharts && s.SkipK3s && s.SkipPkg && !s.SkipImages {
			if s.Quiet {
				log.Infof("Image-only mode with --quiet: exporting all main images for %s", c.ChartName)
			} else {
				mainImages, err := chart.ExtractMainImages(&c, opt)
				if err != nil {
					return err
				}
				selected, err := image.SelectMainImages(c.ChartName, mainImages)
				if err != nil {
					return err
				}
				if selected == nil {
					log.Warnf("Main image selection cancelled for %s; exporting all", c.ChartName)
				} else {
					log.Infof("Selected %d main images for %s", len(selected), c.ChartName)
					selectedMain = selected
				}
			}
		}

		if err := chart.Image(&c, runtimeImages, opt, selectedMain, s.SkipImages, s.SkipRuntimeImages); err != nil {
			return err
		}
	}

	if err := metadata.Save(s); err != nil {
		return err
	}

	log.Infof("Create finished in %s", time.Since(start))
	return nil
}
