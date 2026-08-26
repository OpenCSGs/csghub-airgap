package os

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// GetOSID returns the Linux distribution ID (e.g., "ubuntu", "debian", "centos").
// If it cannot detect the OS, it returns an empty string and an error.
func GetOSID() (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("failed to open /etc/os-release: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id := strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
			return id, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading /etc/os-release: %v", err)
	}

	return "", fmt.Errorf("could not detect OS ID from /etc/os-release")
}

// HasNVIDIAGPU reports whether the host appears to have an NVIDIA GPU
// attached and the kernel driver loaded. Detection is by device file
// presence (/dev/nvidia0, /dev/nvidia1, ...); this is the same heuristic
// used by the NVIDIA device plugin to gate its setup.
//
// On non-Linux hosts, or when the device directory cannot be read (e.g.
// running inside an unprivileged container), the function returns false.
// Callers should treat a false return as "unknown / no GPU" rather than
// as a hard error.
func HasNVIDIAGPU() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	entries, err := filepath.Glob("/dev/nvidia[0-9]*")
	if err != nil {
		return false
	}
	return len(entries) > 0
}
