package pkg

import (
	osi "csghub-airgap/internal/infra/os"
	"csghub-airgap/internal/infra/shell"
	"csghub-airgap/internal/log"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Install installs all previously downloaded NVIDIA toolkit packages from the given destDir.
// It detects the OS, finds the package directory, and installs the packages using the appropriate package manager.
func Install(destDir string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	// Resolve destination directory to absolute path
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of destDir %s: %v", destDir, err)
	}
	packageDir := filepath.Join(absDestDir, "packages")

	id, err := osi.GetOSID()
	if err != nil {
		return err
	}

	switch id {
	// Debian / Ubuntu
	case "ubuntu", "debian":
		// Use dpkg -i to install all .deb files at once
		cmd := fmt.Sprintf("dpkg -i %s/*.deb", packageDir)
		if err := shell.Execute("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to install packages via dpkg: %v", err)
		}

	// RHEL / CentOS / Fedora
	case "rhel", "centos", "almalinux", "fedora":
		pkgMgr := "dnf"
		if _, err := exec.LookPath("dnf"); err != nil {
			pkgMgr = "yum"
		}
		// Install all .rpm files from packageDir using dnf or yum
		rpmFiles := filepath.Join(packageDir, "*.rpm")
		var cmd string
		if pkgMgr == "dnf" {
			cmd = fmt.Sprintf("%s install -y --disablerepo='*' %s", pkgMgr, rpmFiles)
		} else {
			cmd = fmt.Sprintf("%s localinstall -y --disablerepo='*' %s", pkgMgr, rpmFiles)
		}
		if err := shell.Execute("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to install dependencies for package %s: %v. Skipping...", pkgMgr, err)
		}

	// OpenSUSE / SLE
	case "opensuse-leap", "sles", "suse":
		// Install all .rpm files from packageDir using zypper
		cmd := fmt.Sprintf("zypper --non-interactive install -y --no-recommends --repo '' %s/*.rpm", packageDir)
		if err := shell.Execute("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to install dependencies for package %s: %v. Skipping...", cmd, err)
		}

	default:
		// Unknown distro, no-op
	}

	return nil
}
