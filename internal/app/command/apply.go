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

func ApplyOverrides(b *spec.BundleSpec, c *cli.Context) error {
	if !b.SkipK3s {
		b.SkipK3s = c.Bool("skip-k3s")
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
	for i := range b.Charts {
		chart := &b.Charts[i]

		switch chart.Name {
		case "csghub":
			if v := c.String("csghub-version"); v != "" {
				chart.ChartVersion = normalizeVersion(v)
			} else if chart.ChartVersion != "" {
				// version already populated from metadata(deploy)
			} else {
				if latestVersion := hc.LatestVersion(chart); latestVersion != "" {
					chart.ChartVersion = latestVersion
				}
			}

			if domain != "" {
				utils.SetNestedString(chart.Values, domain, "global", "gateway", "external", "domain")
			}

		case "longhorn":
			if v := c.String("longhorn-version"); v != "" {
				chart.ChartVersion = normalizeVersion(v)
			}
			chart.Enabled = !c.Bool("skip-longhorn")
			if data != "" {
				utils.SetNestedString(chart.Values, filepath.Join(data, "longhorn"), "defaultSettings", "defaultDataPath")
			}

		case "nvdp":
			if v := c.String("nvdp-version"); v != "" {
				chart.ChartVersion = normalizeVersion(v)
			}
			chart.Enabled = !c.Bool("skip-nvdp")
		}
	}

	if sets := c.StringSlice("set"); len(sets) > 0 {
		all := flag.ParseFlags(sets)
		for i := range b.Charts {
			chart := &b.Charts[i]

			sub := flag.ParsePrefix(all, chart.Name)
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
	b.Runtime = c.Bool("csghub-runtime")

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
