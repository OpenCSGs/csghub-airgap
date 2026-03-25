package pack

import (
	"bufio"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// PreparePackages sets up NVIDIA container toolkit repository depending on the Linux distro
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
	case "ubuntu", "debian":
		if err := utils.ExecCommand("apt-get", "update"); err != nil {
			return err
		}

		prereqs := []string{"ca-certificates", "curl", "gnupg2", "nfs-common", "open-iscsi"}
		for _, pkg := range prereqs {
			if err := utils.ExecCommand(
				"bash",
				"-c",
				fmt.Sprintf(
					"cd %s && apt-get download -o APT::Sandbox::User=root %s",
					packageDir,
					pkg,
				),
			); err != nil {
				log.Warnf(
					"Failed to download prerequisite package %s: %v",
					pkg,
					err,
				)
			}
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

		version := "1.18.2-1"
		packages := []string{
			fmt.Sprintf("nvidia-container-toolkit=%s", version),
			fmt.Sprintf("nvidia-container-toolkit-base=%s", version),
			fmt.Sprintf("libnvidia-container-tools=%s", version),
			fmt.Sprintf("libnvidia-container1=%s", version),
		}
		for _, pkg := range packages {
			if err := utils.ExecCommand(
				"bash",
				"-c",
				fmt.Sprintf(
					"cd %s && apt-get download -o APT::Sandbox::User=root %s",
					packageDir,
					pkg,
				),
			); err != nil {
				log.Warnf(
					"Failed to download NVIDIA package %s: %v",
					pkg,
					err,
				)
			}
		}

	case "rhel", "centos", "almalinux", "fedora":
		pkgMgr := "dnf"
		useYumDownloader := false
		if _, err := exec.LookPath("dnf"); err != nil {
			pkgMgr = "yum"
			useYumDownloader = true
		}

		// Ensure yum-utils is available for yumdownloader
		if useYumDownloader {
			if err := utils.ExecCommand(
				"bash", "-c",
				fmt.Sprintf("yum install --downloadonly --downloaddir=%s -y yum-utils", packageDir),
			); err != nil {
				log.Warnf("Failed to download yum-utils: %v", err)
			}
		}

		prereqs := []string{"curl", "ca-certificates", "nfs-utils", "iscsi-initiator-utils"}
		for _, pkg := range prereqs {
			if pkgMgr == "dnf" {
				if err := utils.ExecCommand(
					"dnf",
					"download",
					"--resolve",
					"--alldeps",
					"--destdir", packageDir,
					pkg,
				); err != nil {
					log.Warnf(
						"Failed to download prerequisite package %s: %v",
						pkg,
						err,
					)
				}
			} else {
				cmd := fmt.Sprintf(
					"yumdownloader --resolve --destdir=%s %s",
					packageDir,
					pkg,
				)
				if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
					log.Warnf(
						"Failed to download prerequisite package %s: %v",
						pkg,
						err,
					)
				}
			}
		}

		if err := utils.ExecCommand(
			"bash", "-c",
			"curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | "+
				"sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo",
		); err != nil {
			log.Warnf("Failed to add yum repositories for nvidia container toolkit: %v", err)
		}

		version := "1.18.0-1"
		packages := []string{
			fmt.Sprintf("nvidia-container-toolkit-%s", version),
			fmt.Sprintf("nvidia-container-toolkit-base-%s", version),
			fmt.Sprintf("libnvidia-container-tools-%s", version),
			fmt.Sprintf("libnvidia-container1-%s", version),
		}
		for _, pkg := range packages {
			if pkgMgr == "dnf" {
				if err := utils.ExecCommand(
					"dnf",
					"download",
					"--resolve",
					"--alldeps",
					"--destdir", packageDir,
					pkg,
				); err != nil {
					log.Warnf(
						"Failed to download dependencies for package %s: %v. Skipping...",
						pkg,
						err,
					)
				}
			} else {
				cmd := fmt.Sprintf(
					"yum install --downloadonly --downloaddir=%s -y %s",
					packageDir,
					pkg,
				)
				if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
					log.Warnf(
						"Failed to download dependencies for package %s: %v. Skipping...",
						pkg,
						err,
					)
				}
			}
		}

	case "opensuse-leap", "sles", "suse":
		prereqs := []string{"nfs-client", "open-iscsi"}
		for _, pkg := range prereqs {
			if err := utils.ExecCommand(
				"zypper",
				"--non-interactive",
				"download",
				pkg,
			); err != nil {
				log.Warnf(
					"Failed to download prerequisite package %s: %v",
					pkg,
					err,
				)
			}
		}

		if err := utils.ExecCommand(
			"zypper",
			"ar",
			"-f",
			"https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo",
		); err != nil {
			log.Warnf(
				"Failed to add zypper repositories for nvidia container toolkit: %v",
				err,
			)
		}

		version := "1.18.2-1"
		packages := []string{
			fmt.Sprintf("nvidia-container-toolkit-%s", version),
			fmt.Sprintf("nvidia-container-toolkit-base-%s", version),
			fmt.Sprintf("libnvidia-container-tools-%s", version),
			fmt.Sprintf("libnvidia-container1-%s", version),
		}
		for _, pkg := range packages {
			cmd := fmt.Sprintf(
				"zypper --non-interactive install --download-only --auto-agree-with-licenses --download-in-advance --no-recommends -y --pkg-cache-dir %s %s",
				packageDir,
				pkg,
			)
			if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
				log.Warnf(
					"Failed to install dependencies for package %s: %v. Skipping...",
					pkg,
					err,
				)
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
		pkgMgr := "dnf"
		if _, err := exec.LookPath("dnf"); err != nil {
			pkgMgr = "yum"
		}
		// Install all .rpm files from packageDir using dnf or yum
		var cmd string
		if pkgMgr == "dnf" {
			cmd = fmt.Sprintf("dnf install -y %s", filepath.Join(packageDir, "*.rpm"))
		} else {
			cmd = fmt.Sprintf("yum install -y %s", filepath.Join(packageDir, "*.rpm"))
		}
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to install dependencies for package %s: %v. Skipping...", pkgMgr, err)
		}

	// OpenSUSE / SLE
	case "opensuse-leap", "sles", "suse":
		// Install all .rpm files from packageDir using zypper
		cmd := fmt.Sprintf("zypper --non-interactive install -y %s", filepath.Join(packageDir, "*.rpm"))
		if err := utils.ExecCommand("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to install dependencies for package %s: %v. Skipping...", cmd, err)
		}

	default:
		// Unknown distro, no-op
	}

	return nil
}
