package pack

import (
	"bufio"
	"csghub-airgap/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// PreparePackages sets up NVIDIA container toolkit repository depending on the Linux distro
func PreparePackages(destDir string) error {
	if runtime.GOOS != "linux" {
		// Non-Linux OS, nothing to do
		return nil
	}

	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of destDir %s: %v", destDir, err)
	}
	packageDir := filepath.Join(absDestDir, "packages")

	// Ensure apt cache subdirectories exist for apt-get -d install (archives/partial)
	aptArchivesPartial := filepath.Join(packageDir, "archives", "partial")
	if err := os.MkdirAll(aptArchivesPartial, 0755); err != nil {
		return fmt.Errorf("failed to create apt cache subdirectory %s: %v", aptArchivesPartial, err)
	}

	// Detect Linux distribution from /etc/os-release
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return fmt.Errorf("failed to open /etc/os-release: %w", err)
	}
	defer f.Close()

	var id string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Use ExecCommand from utils package to run commands
	switch id {
	// Ubuntu / Debian
	case "ubuntu", "debian":
		// Install prerequisites
		if err := utils.ExecCommand("apt-get", "update"); err != nil {
			return err
		}
		// Download prerequisites packages to packageDir with dependencies
		if err := utils.ExecCommand("apt-get", "-y", "-d", "install", "-o", fmt.Sprintf("dir::cache=%s", packageDir),
			"--download-only", "--reinstall", "--no-install-recommends", "ca-certificates", "curl", "gnupg2", "jq", "nfs-common"); err != nil {
			return err
		}

		// Add NVIDIA GPG key and repository
		if err := utils.ExecCommand("bash", "-c", "curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --yes --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg"); err != nil {
			return err
		}
		if err := utils.ExecCommand("bash", "-c", "curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list"); err != nil {
			return err
		}

		// Update packages list
		if err := utils.ExecCommand("apt-get", "update"); err != nil {
			return err
		}

		// Download NVIDIA Container Toolkit packages to packageDir with dependencies
		version := "1.19.0-1"
		packages := []string{
			fmt.Sprintf("nvidia-container-toolkit=%s", version),
			fmt.Sprintf("nvidia-container-toolkit-base=%s", version),
			fmt.Sprintf("libnvidia-container-tools=%s", version),
			fmt.Sprintf("libnvidia-container1=%s", version),
		}
		for _, pkg := range packages {
			if err := utils.ExecCommand("apt-get", "-y", "-d", "install", "-o", fmt.Sprintf("dir::cache=%s", packageDir),
				"--download-only", "--reinstall", pkg); err != nil {
				return err
			}
		}

	// RHEL / CentOS / Rocky / AlmaLinux / Fedora
	case "rhel", "centos", "almalinux", "fedora":
		// Install prerequisites including nfs-utils
		if err := utils.ExecCommand("dnf", "-y", "install", "curl", "ca-certificates", "jq", "nfs-utils"); err != nil {
			return err
		}

		// Add NVIDIA repository
		if err := utils.ExecCommand("bash", "-c", `curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | \
  sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo`); err != nil {
			return err
		}

		// Download NVIDIA container toolkit packages to packageDir with dependencies
		version := "1.19.0-1"
		packages := []string{
			fmt.Sprintf("nvidia-container-toolkit-%s", version),
			fmt.Sprintf("nvidia-container-toolkit-base-%s", version),
			fmt.Sprintf("libnvidia-container-tools-%s", version),
			fmt.Sprintf("libnvidia-container1-%s", version),
		}
		for _, pkg := range packages {
			if err := utils.ExecCommand("dnf", "download", "--resolve", "--alldeps", "--destdir", packageDir, pkg); err != nil {
				return err
			}
		}

	// OpenSUSE / SLE
	case "opensuse-leap", "sles", "suse":
		// Install prerequisites including nfs-client
		if err := utils.ExecCommand("zypper", "install", "-y", "nfs-client", "jq"); err != nil {
			return err
		}

		// Add NVIDIA repository
		if err := utils.ExecCommand("zypper", "ar", "-f", "https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo"); err != nil {
			return err
		}

		version := "1.19.0-1"
		packages := []string{
			fmt.Sprintf("nvidia-container-toolkit-%s", version),
			fmt.Sprintf("nvidia-container-toolkit-base-%s", version),
			fmt.Sprintf("libnvidia-container-tools-%s", version),
			fmt.Sprintf("libnvidia-container1-%s", version),
		}
		for _, pkg := range packages {
			cmd := fmt.Sprintf("zypper --non-interactive install --download-only --auto-agree-with-licenses --download-in-advance --no-recommends -y --pkg-cache-dir %s %s", packageDir, pkg)
			if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
				return err
			}
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
		// Non-Linux OS, nothing to do
		return nil
	}

	// Resolve destination directory to absolute path
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of destDir %s: %v", destDir, err)
	}
	packageDir := filepath.Join(absDestDir, "packages")

	// Detect Linux distribution from /etc/os-release (same as PreparePackages)
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return fmt.Errorf("failed to open /etc/os-release: %w", err)
	}
	defer f.Close()

	var id string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			id = strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	switch id {
	// Debian / Ubuntu
	case "ubuntu", "debian":
		// Install all .deb files from packageDir/archives using dpkg
		archivesDir := filepath.Join(packageDir, "archives")
		// Use dpkg -i to install all .deb files at once
		cmd := fmt.Sprintf("apt install -y %s", filepath.Join(archivesDir, "*.deb"))
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			// dpkg -i may fail due to missing dependencies, so we proceed to fix them below
			// Return error only if apt-get -f install fails
			// But log the error for dpkg -i
			fmt.Fprintf(os.Stderr, "warning: apt install failed: %v\n", err)
		}
		// Fix broken dependencies
		if err := utils.ExecCommand("bash", "-c", "apt-get -f install -y"); err != nil {
			return fmt.Errorf("apt-get -f install failed: %w", err)
		}

	// RHEL / CentOS / Fedora
	case "rhel", "centos", "almalinux", "fedora":
		// Install all .rpm files from packageDir using dnf
		cmd := fmt.Sprintf("dnf install -y %s", filepath.Join(packageDir, "*.rpm"))
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			return fmt.Errorf("dnf install failed: %w", err)
		}

	// OpenSUSE / SLE
	case "opensuse-leap", "sles", "suse":
		// Install all .rpm files from packageDir using zypper
		cmd := fmt.Sprintf("zypper --non-interactive install -y %s", filepath.Join(packageDir, "*.rpm"))
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			return fmt.Errorf("zypper install failed: %w", err)
		}

	default:
		// Unknown distro, no-op
	}

	return nil
}
