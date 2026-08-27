package chart

import (
	"csghub-airgap/internal/spec"
	"fmt"
	"sort"
	"strings"
)

// HelmUpgradeCmd renders the `helm upgrade --install` command that mirrors what
// the Helm SDK's Install/Upgrade actions run, so --dry-run can print a
// copy-pasteable command instead of a summary. The single upgrade --install form
// covers both a fresh install and an upgrade of an existing release.
func HelmUpgradeCmd(c *spec.ChartSpec) string {
	args := []string{"helm", "upgrade", "--install", c.RepoName, c.ChartFilePath()}
	args = append(args, "--namespace", c.RepoName, "--create-namespace")
	if c.ChartVersion != "" {
		args = append(args, "--version", c.ChartVersion)
	}
	if kubeconfig, err := DiscoverKubeConfig(); err == nil {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	for _, kv := range flattenValues(c.Values) {
		args = append(args, "--set", kv)
	}
	return strings.Join(args, " ")
}

// HelmPullCmd renders the `helm pull` command that mirrors what chart.Pull runs.
func HelmPullCmd(c *spec.ChartSpec) string {
	args := []string{"helm", "pull", c.ChartRef()}
	if c.ChartVersion != "" {
		args = append(args, "--version", c.ChartVersion)
	}
	args = append(args, "--destination", c.ChartDir())
	return strings.Join(args, " ")
}

// flattenValues turns nested Helm values into --set style key=value pairs.
// Nested maps become dot-separated keys, lists become {a,b} (Helm's list
// syntax), nil becomes "null". Keys are sorted so the output is deterministic.
func flattenValues(v map[string]interface{}) []string {
	var out []string
	var walk func(prefix string, val interface{})
	walk = func(prefix string, val interface{}) {
		switch t := val.(type) {
		case map[string]interface{}:
			keys := make([]string, 0, len(t))
			for k := range t {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				key := k
				if prefix != "" {
					key = prefix + "." + k
				}
				walk(key, t[k])
			}
		case map[interface{}]interface{}:
			byKey := make(map[string]interface{}, len(t))
			keys := make([]string, 0, len(t))
			for k, sub := range t {
				ks := fmt.Sprintf("%v", k)
				keys = append(keys, ks)
				byKey[ks] = sub
			}
			sort.Strings(keys)
			for _, k := range keys {
				key := k
				if prefix != "" {
					key = prefix + "." + k
				}
				walk(key, byKey[k])
			}
		case []interface{}:
			var elems []string
			for _, e := range t {
				elems = append(elems, fmt.Sprintf("%v", e))
			}
			out = append(out, prefix+"={"+strings.Join(elems, ",")+"}")
		case nil:
			out = append(out, prefix+"=null")
		default:
			out = append(out, fmt.Sprintf("%s=%v", prefix, val))
		}
	}
	walk("", v)
	return out
}
