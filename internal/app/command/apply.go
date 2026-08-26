package command

import (
	hc "csghub-airgap/internal/chart"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/infra/net"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"csghub-airgap/internal/utils/flag"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
)

func normalizeVersion(v string) string {
	return strings.TrimPrefix(v, "v")
}

// skipSet is the parsed result of the unified --skip flag. ChartNames are
// chart manifest names (csghub, longhorn, nvdp) to disable; the bools are
// the coarse-grained component skips.
type skipSet struct {
	K3s           bool
	Charts        bool
	Images        bool
	RuntimeImages bool
	Pkg           bool
	ChartNames    []string
}

// parseSkips normalizes the comma-separated --skip values into a skipSet.
// Accepted values (case-insensitive, commas and repeated flags both work):
//
//	k3s          skip k3s download/install
//	charts       skip chart pull (reuse existing chart files) but still
//	             export main images
//	image        skip ALL image pull (main chart images and runtime images)
//	runtime      skip only the user-selected runtime images
//	package      skip NVIDIA container toolkit package download/install
//	<chart name> skip that specific chart entirely (pull + its images)
func parseSkips(values []string) skipSet {
	var sk skipSet
	seen := make(map[string]struct{})

	// Flatten repeated flags and any embedded commas, lowercased + trimmed.
	var tokens []string
	for _, v := range values {
		for _, t := range strings.Split(v, ",") {
			if n := strings.ToLower(strings.TrimSpace(t)); n != "" {
				tokens = append(tokens, n)
			}
		}
	}

	for _, t := range tokens {
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}

		switch t {
		case "k3s":
			sk.K3s = true
		case "charts":
			sk.Charts = true
		case "image":
			sk.Images = true
			sk.RuntimeImages = true
		case "runtime":
			sk.RuntimeImages = true
		case "package":
			sk.Pkg = true
		default:
			// Treat as a chart name; unmatched names are warned on later.
			sk.ChartNames = append(sk.ChartNames, t)
		}
	}
	return sk
}

func ApplyOverrides(b *spec.BundleSpec, c *cli.Context) error {
	sk := parseSkips(c.StringSlice("skip"))
	if sk.K3s {
		b.SkipK3s = true
	}
	if sk.Charts {
		b.SkipCharts = true
	}
	if sk.Images {
		b.SkipImages = true
	}
	if sk.RuntimeImages {
		b.SkipRuntimeImages = true
	}
	if sk.Pkg {
		b.SkipPkg = true
	}

	if b.K3s == nil {
		return fmt.Errorf("invalid bundle spec: k3s config is nil")
	}

	data := c.String("data")
	domain := c.String("csghub-domain")

	if data != "" {
		b.K3s.InstallArgs["--data-dir"] = filepath.Join(data, "k3s")
	}

	for _, kv := range c.StringSlice("k3s-env") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if b.K3s.InstallEnvs == nil {
			b.K3s.InstallEnvs = make(map[string]string)
		}
		b.K3s.InstallEnvs[parts[0]] = parts[1]
	}

	for _, kv := range c.StringSlice("k3s-arg") {
		parts := strings.SplitN(kv, "=", 2)
		if b.K3s.InstallArgs == nil {
			b.K3s.InstallArgs = make(map[string]string)
		}

		key := parts[0]
		if !strings.HasPrefix(key, "--") {
			key = "--" + key
		}

		if len(parts) == 2 {
			b.K3s.InstallArgs[key] = parts[1]
		} else {
			b.K3s.InstallArgs[key] = ""
		}
	}

	// ===== charts =====
	// Track which --skip chart names actually matched a chart in the
	// manifest so typos and stale names are surfaced instead of silently
	// ignored.
	matchedChartNames := make(map[string]struct{})
	for i := range b.Charts {
		chart := &b.Charts[i]

		// --skip <chartname> disables that chart.
		for _, name := range sk.ChartNames {
			if chart.Name == name {
				chart.Enabled = false
				matchedChartNames[name] = struct{}{}
				log.Debugf("Chart %s disabled via --skip %s", chart.Name, name)
			}
		}

		switch chart.Name {
		case "csghub":
			if chart.ChartVersion == "" {
				if latestVersion := hc.LatestVersion(chart); latestVersion != "" {
					chart.ChartVersion = latestVersion
				}
			}

			if domain != "" {
				utils.SetNestedString(chart.Values, domain, "global", "gateway", "external", "domain")
			}

		case "longhorn":
			if data != "" {
				utils.SetNestedString(chart.Values, filepath.Join(data, "longhorn"), "defaultSettings", "defaultDataPath")
			}
		}
	}

	// Surface --skip chart names that did not match any chart in the manifest.
	for _, name := range sk.ChartNames {
		if _, ok := matchedChartNames[name]; !ok {
			log.Warnf("--skip %s: no chart named %q in the bundle; ignoring", name, name)
		}
	}

	// --set <chart>.key=value overrides the chart's Helm values, except the
	// special key `version` which maps to the chart's ChartVersion instead.
	if sets := c.StringSlice("set"); len(sets) > 0 {
		all := flag.ParseFlags(sets)
		for i := range b.Charts {
			chart := &b.Charts[i]

			sub := flag.ParsePrefix(all, chart.Name)
			if len(sub) == 0 {
				continue
			}

			if v, ok := sub["version"]; ok {
				if ver, ok := v.(string); ok && ver != "" {
					chart.ChartVersion = normalizeVersion(ver)
					log.Debugf("Chart %s version set via --set %s.version=%s", chart.Name, chart.Name, ver)
				}
				delete(sub, "version")
			}

			if len(sub) == 0 {
				continue
			}
			if chart.Values == nil {
				chart.Values = map[string]interface{}{}
			}

			Merge(chart.Values, sub)
		}
	}

	b.Arch = config.DefaultArchitecture

	// On deploy, replace a --flannel-iface recorded from the build machine
	// (e.g. en0 on macOS) with the current host's default interface, unless
	// the user explicitly overrode it via --k3s-arg. This lets bundles created
	// cross-platform deploy correctly when the target host uses a different
	// NIC name (eth0, ens33, ...).
	if c.Command.Name == "deploy" && !c.IsSet("k3s-arg") {
		if _, ok := b.K3s.InstallArgs["--flannel-iface"]; ok {
			iface, err := net.GetDefaultInterface()
			if err == nil && iface != "" {
				b.K3s.InstallArgs["--flannel-iface"] = iface
				log.Debugf("Overriding --flannel-iface with host interface %s", iface)
			}
		}
	}

	return nil
}

func Merge(dst, src map[string]interface{}) {
	for k, v := range src {
		if vMap, ok := v.(map[string]interface{}); ok {
			if dst[k] == nil {
				dst[k] = map[string]interface{}{}
			}
			if dstMap, ok := dst[k].(map[string]interface{}); ok {
				Merge(dstMap, vMap)
			} else {
				dst[k] = vMap
			}
		} else {
			dst[k] = v
		}
	}
}
