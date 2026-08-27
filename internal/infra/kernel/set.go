package kernel

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/infra/shell"
	"csghub-airgap/internal/log"
	"fmt"
	"os"
	"sort"
	"strings"
)

var (
	DefaultKernelParams = map[string]string{
		"fs.inotify.max_user_instances": "512",
	}

	DefaultKernelConfFile = "/etc/sysctl.d/10-k3s.conf"
)

// SetParams merges user params with default kernel params (user params take precedence),
// writes to system config file, and applies settings immediately with sysctl -p.
// Changes persist after reboot.
func SetParams(params map[string]string) error {
	log.Info("Setting kernel sysctl parameters")

	// Merge default parameters with provided params, giving priority to params.
	merged := make(map[string]string)
	// Apply defaults first
	for k, v := range DefaultKernelParams {
		merged[k] = v
	}
	// Override with user provided params
	for k, v := range params {
		merged[k] = v
	}

	if config.DryRun {
		var lines []string
		keys := make([]string, 0, len(merged))
		for k := range merged {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			lines = append(lines, fmt.Sprintf("%s = %s", k, merged[k]))
		}
		log.Infof("[DRY-RUN] $ printf '%s\\n' > %s", strings.Join(lines, `\n`), DefaultKernelConfFile)
		log.Infof("[DRY-RUN] $ sysctl -p %s", DefaultKernelConfFile)
		return nil
	}

	// Open the configuration file for write, create if it doesn't exist, truncate existing.
	file, err := os.OpenFile(DefaultKernelConfFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open kernel config file: %v", err)
	}
	defer file.Close()

	// Write each key=value pair to the configuration file.
	for key, val := range merged {
		_, err = fmt.Fprintf(file, "%s = %s\n", key, val)
		if err != nil {
			return fmt.Errorf("failed to write kernel config: %v", err)
		}
	}

	log.Info("Applying kernel parameters with sysctl")
	if err := shell.Execute("sysctl", "-p", DefaultKernelConfFile); err != nil {
		return fmt.Errorf("failed to apply kernel parameters: %v", err)
	}

	log.Info("Kernel parameters applied successfully")
	return nil
}
