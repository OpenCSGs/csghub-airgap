package kernel

import (
	"csghub-airgap/internal/log"
	"fmt"
	"os"
	"os/exec"
)

var (
	DefaultKernelParams = map[string]string{
		"fs.inotify.max_user_instances": "512",
	}
	DefaultKernelConfFile = "/etc/sysctl.d/10-k3s.conf"
)

// SetSysctl merges the provided params with the default kernel parameters,
// giving priority to the provided params, writes them to the default configuration file,
// and applies the changes immediately using "sysctl -p".
// This ensures the kernel parameters persist across reboots and take effect immediately.
func SetSysctl(params map[string]string) error {
	log.Infof("Setting sysctl kernel params: %v", params)

	// Merge default parameters with provided params, giving priority to params.
	mergedParams := make(map[string]string)
	for k, v := range DefaultKernelParams {
		mergedParams[k] = v
	}
	for k, v := range params {
		mergedParams[k] = v
	}

	// Open the configuration file for write, create if it doesn't exist, truncate existing.
	f, err := os.OpenFile(DefaultKernelConfFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write each key=value pair to the configuration file.
	for key, value := range mergedParams {
		line := fmt.Sprintf("%s = %s\n", key, value)
		if _, err := f.WriteString(line); err != nil {
			return err
		}
	}

	// Apply the new sysctl settings immediately.
	cmd := exec.Command("sysctl", "-p", DefaultKernelConfFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
