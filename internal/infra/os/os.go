package os

import (
	"bufio"
	"fmt"
	"os"
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
