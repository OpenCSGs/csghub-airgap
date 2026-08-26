package bundle

import (
	"context"
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

		// Export the chart's own images first; the runtime images (csghub only)
		// are decided and exported afterwards once the main images are done.
		if err := chart.Image(&c, nil, opt, selectedMain, s.SkipImages, true); err != nil {
			return err
		}

		// Runtime images belong to the csghub chart only. Prompt for them AFTER
		// the main chart images are exported so the interactive selection does
		// not sit at the very start of the run.
		if c.RepoName == "csghub" {
			if err := s.selectRuntimeImages(); err != nil {
				return err
			}
			if !s.SkipRuntimeImages && len(s.SelectedRuntimeImages) > 0 {
				if err := chart.Image(&c, s.SelectedRuntimeImages, opt, nil, true, false); err != nil {
					return err
				}
			}
		}
	}

	if err := metadata.Save(s); err != nil {
		return err
	}

	log.Infof("Create finished in %s", time.Since(start))
	return nil
}

// selectRuntimeImages prompts the user (TUI) to pick which runtime images to
// bundle for the csghub chart. Runs after the main chart images are exported.
// No-op when runtime selection is disabled (--skip runtime/image) or when
// --quiet suppresses interactive prompts. An empty TUI selection (or pressing
// q/esc) also skips runtime images. The result is persisted to metadata.yaml
// by the caller's metadata.Save.
func (s *Spec) selectRuntimeImages() error {
	switch {
	case s.SkipRuntimeImages:
		log.Infof("Skipping runtime image selection (--skip runtime/image)")
		return nil
	case s.Quiet:
		log.Infof("Skipping runtime image selection (--quiet)")
		return nil
	}

	if !log.IsDebug() {
		log.Infof("Fetching available runtime images...")
	}
	selected, err := image.SelectRuntimeImages(context.Background(), image.SourceGitHub, "main")
	if err != nil {
		return err
	}
	s.SelectedRuntimeImages = selected
	switch {
	case selected == nil:
		log.Warn("Runtime image selection cancelled; skipping runtime images")
	case len(selected) == 0:
		log.Warn("No runtime images selected; skipping runtime image export")
	default:
		log.Infof("Selected %d runtime images", len(selected))
	}
	return nil
}
