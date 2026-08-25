package os

// Service control helpers wrapping systemctl. All functions assume Linux;
// callers should gate on runtime.GOOS == "linux" if they need to skip
// non-Linux hosts.

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// systemctlLookPath returns the path to systemctl, preferring the system
// binary but falling back to /bin/systemctl for environments where PATH
// is restricted (e.g. some container runtimes).
func systemctlLookPath() (string, error) {
	if p, err := exec.LookPath("systemctl"); err == nil {
		return p, nil
	}
	for _, p := range []string{"/usr/bin/systemctl", "/bin/systemctl"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("systemctl not found on PATH or in /usr/bin or /bin")
}

// IsServiceActive reports whether the named systemd unit is currently
// active. Returns false on hosts where systemctl is unavailable or when
// the unit is not loaded.
func IsServiceActive(name string) (bool, error) {
	bin, err := systemctlLookPath()
	if err != nil {
		return false, err
	}
	cmd := exec.Command(bin, "is-active", name)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return true, nil
	}
	trimmed := strings.TrimSpace(string(out))
	return trimmed == "active", nil
}

// StopService stops a systemd unit.
func StopService(name string) error {
	bin, err := systemctlLookPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "stop", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl stop %s: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// DisableService prevents a systemd unit from starting at boot.
func DisableService(name string) error {
	bin, err := systemctlLookPath()
	if err != nil {
		return err
	}
	cmd := exec.Command(bin, "disable", name)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl disable %s: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}
