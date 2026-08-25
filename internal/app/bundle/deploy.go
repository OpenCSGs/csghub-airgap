package bundle

import (
	"csghub-airgap/internal/chart"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/infra/kernel"
	"csghub-airgap/internal/infra/os"
	"csghub-airgap/internal/infra/pkg"
	"csghub-airgap/internal/k3s"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"fmt"
	"strings"
	"time"
)

func (s *Spec) Deploy() error {
	start := time.Now()

	if err := os.CheckOS(); err != nil {
		return err
	}

	if err := os.CheckArch(); err != nil {
		return err
	}

	// Auto-disable nvdp (and the NVIDIA toolkit packages that feed it) when
	// the host has no NVIDIA GPU. This keeps the default `deploy` behaviour
	// safe on non-GPU clusters even though --skip-nvdp now defaults to
	// false (i.e. "install"). Users who want to force-install anyway can
	// re-enable nvdp.Enabled in the bundle's metadata.yaml by hand.
	if !os.HasNVIDIAGPU() {
		for i := range s.Charts {
			if s.Charts[i].ChartName == "nvidia-device-plugin" && s.Charts[i].Enabled {
				log.Warnf("No NVIDIA GPU detected on this host; auto-disabling nvdp chart")
				s.Charts[i].Enabled = false
			}
		}
	}

	if !nvdpEnabled(s.Charts) {
		log.Infof("Skipping NVIDIA toolkit package install (nvdp chart disabled)")
	} else if err := pkg.InstallFrom(config.DefaultBaseDir); err != nil {
		return err
	}

	if err := kernel.SetParams(nil); err != nil {
		return err
	}

	if !s.SkipK3s {
		if err := k3s.Install(s.K3s); err != nil {
			return err
		}
	}

	for _, c := range s.Charts {
		if !c.Enabled {
			log.Debugf("Skipping disabled chart %s", c.ChartName)
			continue
		}
		imageSpec := &spec.ImageSpec{
			Images:       nil,
			ChartName:    c.ChartName,
			ChartVersion: c.ChartVersion,
		}
		if err := image.Import(imageSpec); err != nil {
			return err
		}

		if s.Runtime {
			imageSpec := &spec.ImageSpec{
				Images:       nil,
				ChartName:    fmt.Sprintf("%s-%s", c.ChartName, "runtime"),
				ChartVersion: c.ChartVersion,
			}
			if err := image.Import(imageSpec); err != nil {
				return err
			}
		}

		if chart.List(&c) {
			if err := chart.Upgrade(&c); err != nil {
				return err
			}
		} else {
			if err := chart.Install(&c); err != nil {
				return err
			}
		}
	}

	var domain string
	for i := range s.Charts {
		c := &s.Charts[i]
		if c.ChartName == "csghub" {
			domain, _ = utils.GetNestedString(c.Values, "global", "gateway", "external", "domain")
			break
		}
	}

	// Normalize domain:
	// - If more than 2 labels (e.g. a.b.example.com), keep last two → example.com
	// - If 2 or fewer (e.g. example.com), keep as is
	parts := strings.Split(domain, ".")
	if len(parts) > 2 {
		domain = strings.Join(parts[len(parts)-2:], ".")
	}

	if err := k3s.PatchInstall(s.K3s, domain); err != nil {
		return err
	}

	log.Warn("Use 'kubectl get pods -n csghub' to check if service is ready")

	log.Infof("Deploy finished in %s", time.Since(start))

	return nil
}

// nvdpEnabled returns true if the nvdp chart (nvidia-device-plugin) is set
// to install in this bundle. Used to gate NVIDIA toolkit package download
// and install: when --skip-nvdp is in effect (default for `deploy`) the
// package operations are skipped too, since the toolkit libraries have no
// consumer.
func nvdpEnabled(charts []spec.ChartSpec) bool {
	for _, c := range charts {
		if c.ChartName == "nvidia-device-plugin" {
			return c.Enabled
		}
	}
	return false
}
