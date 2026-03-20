package bundle

import (
	"fmt"
	"runtime"
)

func checkArch() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("unsupported platform")
	}

	switch runtime.GOARCH {
	case "amd64", "arm64":
		// supported
	default:
		return fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	return nil
}
