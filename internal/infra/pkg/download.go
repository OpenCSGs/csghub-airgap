package pkg

import (
	osi "csghub-airgap/internal/infra/os"
	"csghub-airgap/internal/infra/shell"
	"csghub-airgap/internal/infra/user"
	"csghub-airgap/internal/log"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const NvidiaContainerToolkitVersion = "1.18.0-1"

// Download locations for when packages cannot be fetched automatically
// (cross-architecture download or non-Linux hosts).
const (
	InstallGuideURL = "https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html"
	DebianRepoURL   = "https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list"
	RpmRepoURL      = "https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo"
)

var packages = []string{
	fmt.Sprintf("nvidia-container-toolkit=%s", NvidiaContainerToolkitVersion),
	fmt.Sprintf("nvidia-container-toolkit-base=%s", NvidiaContainerToolkitVersion),
	fmt.Sprintf("libnvidia-container-tools=%s", NvidiaContainerToolkitVersion),
	fmt.Sprintf("libnvidia-container1=%s", NvidiaContainerToolkitVersion),
}

// PackageList returns the NVIDIA packages (with pinned versions) that a bundle needs.
func PackageList() string {
	return strings.Join(packages, " ")
}

// Download sets up NVIDIA container toolkit repository depending on the Linux distro
func Download(destDir string) error {
	if runtime.GOOS != "linux" {
		log.Warnf("Cannot download NVIDIA container toolkit packages on %s (Linux required)", runtime.GOOS)
		log.Warnf("Download them manually, e.g. apt/dnf/zypper install %s", PackageList())
		log.Warnf("Debian/Ubuntu repo: %s", DebianRepoURL)
		log.Warnf("RHEL/Fedora/openSUSE repo: %s", RpmRepoURL)
		log.Warnf("See install guide: %s", InstallGuideURL)
		return nil
	}

	if err := user.EnsureRoot(); err != nil {
		return err
	}

	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path of destDir %s: %v", destDir, err)
	}

	packageDir := filepath.Join(absDestDir, "packages")
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return fmt.Errorf("failed to create packageDir %s: %v", packageDir, err)
	}

	id, err := osi.GetOSID()
	if err != nil {
		return err
	}

	// Strict cache check: skip the apt/dnf/zypper dance when the package dir
	// already contains a complete, intact, dependency-resolving set of
	// packages. Any validation failure auto-clears the lock file and falls
	// through to a fresh download.
	if !NoCache {
		if ok, reason := validateCache(packageDir); ok {
			log.Infof("Reusing cached NVIDIA packages (%s), skipping download", reason)
			return nil
		} else if reason != "" {
			log.Warnf("Cached NVIDIA packages invalid (%s), re-downloading", reason)
			_ = os.Remove(filepath.Join(packageDir, lockFileName))
		}
	}

	switch id {
	case "ubuntu", "debian":
		if err := shell.Execute("apt-get", "update"); err != nil {
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

		if err := shell.Execute("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to download prerequisite packages and dependencies: %v", err)
		}

		if err := shell.Execute(
			"bash", "-c",
			"curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | "+
				"sudo gpg --yes --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg",
		); err != nil {
			log.Warnf("Failed to add gpgkey for nvidia package registry: %v", err)
		}

		if err := shell.Execute(
			"bash", "-c",
			"curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | "+
				"sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | "+
				"sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list",
		); err != nil {
			log.Warnf("Failed to add apt repositories for nvidia container toolkit: %v", err)
		}

		if err := shell.Execute("apt-get", "update"); err != nil {
			log.Warnf("Failed to update apt cache after adding NVIDIA repo: %v", err)
		}

		pkgStr := strings.Join(packages, " ")
		cmd = fmt.Sprintf(
			"apt-get install --download-only --no-install-recommends --allow-downgrades -y -o APT::Sandbox::User=root "+
				"-o Dir::Cache=%s -o Dir::Cache::archives=%s %s",
			packageDir,
			packageDir,
			pkgStr,
		)

		if err := shell.Execute("bash", "-c", cmd); err != nil {
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
			if err := shell.Execute("bash", "-c", cmd); err != nil {
				log.Warnf("Failed to download prerequisite packages via dnf: %v", err)
			}
		} else {
			cmd := fmt.Sprintf("yum install --downloadonly --downloaddir=%s -y %s", packageDir, prereqStr)
			if err := shell.Execute("bash", "-c", cmd); err != nil {
				log.Warnf("Failed to download prerequisite packages via yum: %v", err)
			}
		}

		if err := shell.Execute(
			"bash", "-c",
			"curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | "+
				"sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo",
		); err != nil {
			log.Warnf("Failed to add yum repositories for nvidia container toolkit: %v", err)
		}

		pkgStr := strings.ReplaceAll(strings.Join(packages, " "), "=", "-")
		if pkgMgr == "dnf" {
			cmd := fmt.Sprintf("dnf download --resolve --alldeps --destdir=%s %s", packageDir, pkgStr)
			if err := shell.Execute("bash", "-c", cmd); err != nil {
				log.Warnf("Failed to download NVIDIA packages via dnf: %v", err)
			}
		} else {
			cmd := fmt.Sprintf("yum install --downloadonly --downloaddir=%s -y %s", packageDir, pkgStr)
			if err := shell.Execute("bash", "-c", cmd); err != nil {
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
		if err := shell.Execute("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to download prerequisite packages via zypper: %v", err)
		}

		if err := shell.Execute(
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
		if err := shell.Execute("bash", "-c", cmd); err != nil {
			log.Warnf("Failed to download NVIDIA packages via zypper: %v", err)
		}

	default:
		// Unknown distro, no-op
	}

	// Persist a cache lock so the next `airgap bundle create` can skip this
	// whole flow when the package set is unchanged.
	if !NoCache {
		if err := writeLock(packageDir, id, sourceRepoForDistro(id)); err != nil {
			log.Warnf("Failed to write package lock file: %v", err)
		} else {
			log.Infof("Wrote package cache lock: %s", filepath.Join(packageDir, lockFileName))
		}
	}

	return nil
}

// sourceRepoForDistro returns the upstream repo URL that was used for the
// given distro, recorded in the lock file so future audits can verify the
// source of every cached package.
func sourceRepoForDistro(id string) string {
	switch id {
	case "ubuntu", "debian":
		return DebianRepoURL
	case "rhel", "centos", "almalinux", "fedora", "opensuse-leap", "sles", "suse":
		return RpmRepoURL
	default:
		return ""
	}
}
