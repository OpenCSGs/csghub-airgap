package os

import (
	"fmt"
	"runtime"
)

func CheckArch() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported platform: %v", runtime.GOOS)
	}

	switch runtime.GOARCH {
	case "amd64", "arm64":
		// supported
	default:
		return fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	return nil
}
