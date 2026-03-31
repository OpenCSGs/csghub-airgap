package os

import (
	"errors"
	"fmt"
	"runtime"
)

func CheckArch() error {
	if runtime.GOOS != "linux" {
		return errors.New("unsupported platform")
	}

	switch runtime.GOARCH {
	case "amd64", "arm64":
		// supported
	default:
		return fmt.Errorf("unsupported architecture: %s", runtime.GOARCH)
	}

	return nil
}
