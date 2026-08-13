package os

import (
	"fmt"
	"runtime"
)

// CheckArch validates the current CPU architecture. It is OS-independent so
// bundles can be created from any host (e.g. macOS).
func CheckArch() error {
	switch runtime.GOARCH {
	case "amd64", "arm64":
		return nil
	default:
		return fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}
}

// CheckOS validates the current OS. Deploy is Linux-only because it requires
// systemd, kernel params, and distro package managers.
func CheckOS() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported platform: %v", runtime.GOOS)
	}
	return nil
}
