package pack

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const NvidiaContainerToolkitVersion = "1.18.0-1"

var packages = []string{
	fmt.Sprintf("nvidia-container-toolkit=%s", NvidiaContainerToolkitVersion),
	fmt.Sprintf("nvidia-container-toolkit-base=%s", NvidiaContainerToolkitVersion),
	fmt.Sprintf("libnvidia-container-tools=%s", NvidiaContainerToolkitVersion),
	fmt.Sprintf("libnvidia-container1=%s", NvidiaContainerToolkitVersion),
}

// PreparePackages sets up NVIDIA container toolkit repository depending on the Linux distro
func PreparePackages(destDir string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of destDir %s: %v", destDir, err)
	}

	packageDir := filepath.Join(absDestDir, "packages")
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create packageDir %s: %v", packageDir, err)
	}

	id, err := utils.GetOSID()
	if err != nil {
		return err
	}

	switch id {
	case "ubuntu", "debian":
		if err := utils.ExecCommand("apt-get", "update"); err != nil {
			return err
		}

		prereqs := []string{"ca-certificates", "curl", "gnupg2", "nfs-common", "open-iscsi"}
		prereqStr := strings.Join(prereqs, " ")
		cmd := fmt.Sprintf(
			"apt-get install --download-only --no-install-recommends -y -o APT::Sandbox::User=root "+
				"-o Dir::Cache=%s -o Dir::Cache::archives=%s %s",
			packageDir,
			packageDir,
			prereqStr,
		)

		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to download prerequisite packages and dependencies: %v", err)
		}

		if err := utils.ExecCommand(
			"bash", "-c",
			"curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | "+
				"sudo gpg --yes --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg",
		); err != nil {
			log.Warnf("Failed to add gpgkey for nvidia package registry: %v", err)
		}

		if err := utils.ExecCommand(
			"bash", "-c",
			"curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | "+
				"sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | "+
				"sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list",
		); err != nil {
			log.Warnf("Failed to add apt repositories for nvidia container toolkit: %v", err)
		}

		if err := utils.ExecCommand("apt-get", "update"); err != nil {
			log.Warnf("Failed to update apt cache after adding NVIDIA repo: %v", err)
		}

		pkgStr := strings.Join(packages, " ")
		cmd = fmt.Sprintf(
			"apt-get install --download-only --no-install-recommends -y -o APT::Sandbox::User=root "+
				"-o Dir::Cache=%s -o Dir::Cache::archives=%s %s",
			packageDir,
			packageDir,
			pkgStr,
		)

		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to download NVIDIA packages and dependencies: %v", err)
		}

	case "rhel", "centos", "almalinux", "fedora":
		pkgMgr := "dnf"
		if _, err := exec.LookPath("dnf"); err != nil {
			pkgMgr = "yum"
		}

		prereqs := []string{"curl", "ca-certificates", "nfs-utils", "iscsi-initiator-utils"}
		prereqStr := strings.Join(prereqs, " ")
		if pkgMgr == "dnf" {
			cmd := fmt.Sprintf("dnf download --resolve --alldeps --destdir=%s %s", packageDir, prereqStr)
			if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
				log.Warnf("Failed to download prerequisite packages via dnf: %v", err)
			}
		} else {
			cmd := fmt.Sprintf("yum install --downloadonly --downloaddir=%s -y %s", packageDir, prereqStr)
			if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
				log.Warnf("Failed to download prerequisite packages via yum: %v", err)
			}
		}

		if err := utils.ExecCommand(
			"bash", "-c",
			"curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | "+
				"sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo",
		); err != nil {
			log.Warnf("Failed to add yum repositories for nvidia container toolkit: %v", err)
		}

		pkgStr := strings.ReplaceAll(strings.Join(packages, " "), "=", "-")
		if pkgMgr == "dnf" {
			cmd := fmt.Sprintf("dnf download --resolve --alldeps --destdir=%s %s", packageDir, pkgStr)
			if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
				log.Warnf("Failed to download NVIDIA packages via dnf: %v", err)
			}
		} else {
			cmd := fmt.Sprintf("yum install --downloadonly --downloaddir=%s -y %s", packageDir, pkgStr)
			if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
				log.Warnf("Failed to download NVIDIA packages via yum: %v", err)
			}
		}

	case "opensuse-leap", "sles", "suse":
		prereqs := []string{"nfs-client", "open-iscsi"}
		prereqStr := strings.Join(prereqs, " ")

		cmd := fmt.Sprintf(
			"zypper --non-interactive install --download-only --auto-agree-with-licenses --download-in-advance --no-recommends --pkg-cache-dir %s %s",
			packageDir,
			prereqStr,
		)
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to download prerequisite packages via zypper: %v", err)
		}

		if err := utils.ExecCommand(
			"zypper",
			"ar",
			"-f",
			"https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo",
			"nvidia",
		); err != nil {
			log.Warnf(
				"Failed to add zypper repositories for nvidia container toolkit: %v",
				err,
			)
		}

		pkgStr := strings.ReplaceAll(strings.Join(packages, " "), "=", "-")
		cmd = fmt.Sprintf(
			"zypper --non-interactive install --download-only --auto-agree-with-licenses --download-in-advance --no-recommends --pkg-cache-dir %s %s",
			packageDir,
			pkgStr,
		)
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to download NVIDIA packages via zypper: %v", err)
		}

	default:
		// Unknown distro, no-op
	}

	return nil
}

// InstallPackages installs all previously downloaded NVIDIA toolkit packages from the given destDir.
// It detects the OS, finds the package directory, and installs the packages using the appropriate package manager.
func InstallPackages(destDir string) error {
	if runtime.GOOS != "linux" {
		return nil
	}

	// Resolve destination directory to absolute path
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of destDir %s: %v", destDir, err)
	}
	packageDir := filepath.Join(absDestDir, "packages")

	id, err := utils.GetOSID()
	if err != nil {
		return err
	}

	switch id {
	// Debian / Ubuntu
	case "ubuntu", "debian":
		// Use dpkg -i to install all .deb files at once
		cmd := fmt.Sprintf("dpkg -i %s/*.deb", packageDir)
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
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
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to install dependencies for package %s: %v. Skipping...", pkgMgr, err)
		}

	// OpenSUSE / SLE
	case "opensuse-leap", "sles", "suse":
		// Install all .rpm files from packageDir using zypper
		cmd := fmt.Sprintf("zypper --non-interactive install -y --no-recommends --repo '' %s/*.rpm", packageDir)
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to install dependencies for package %s: %v. Skipping...", cmd, err)
		}

	default:
		// Unknown distro, no-op
	}

	return nil
}
