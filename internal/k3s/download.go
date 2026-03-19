package k3s

import (
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

// DownloadK3S Downloads and prepares all required K3s files for offline deployment
func DownloadK3S(spec *spec.K3sSpec) error {
	// Validate input parameters before processing
	if err := spec.Validate(); err != nil {
		return err
	}

	// Download K3s binary
	if err := downloadBinary(spec); err != nil {
		return fmt.Errorf("failed to download K3s binary: %w", err)
	}

	// Download K3s airgap image archive
	if err := downloadImages(spec); err != nil {
		return fmt.Errorf("failed to download K3s images: %w", err)
	}

	// Download K3s installation script
	if err := downloadInstallScript(spec); err != nil {
		return fmt.Errorf("failed to download K3s install script: %w", err)
	}

	return nil
}

// downloadBinary Fetches the K3s binary executable from official release
func downloadBinary(spec *spec.K3sSpec) error {
	// Construct download URL for K3s binary
	downloadURL, _ := url.JoinPath(spec.ReleaseURL, "download", spec.Version, "k3s")
	// Clean path to prevent directory traversal vulnerabilities
	savePath := filepath.Clean(filepath.Join(spec.DestDir, "k3s", "bin", "k3s"))

	// Download file using utility function
	if err := utils.DownloadFile(downloadURL, savePath); err != nil {
		return fmt.Errorf("file download failed [url=%s, path=%s]: %w", downloadURL, savePath, err)
	}

	// Set executable permission for binary file
	if err := os.Chmod(savePath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission [path=%s]: %w", savePath, err)
	}

	return nil
}

// downloadImages Fetches the K3s airgap image archive for offline deployment
func downloadImages(spec *spec.K3sSpec) error {
	// Construct image archive filename
	imgFileName := fmt.Sprintf("k3s-airgap-images-%s.tar.zst", spec.Arch)
	// Construct full download URL
	downloadURL, _ := url.JoinPath(spec.ReleaseURL, "download", spec.Version, imgFileName)
	// Clean and construct save path
	savePath := filepath.Clean(filepath.Join(spec.DestDir, "k3s", "images", imgFileName))

	// Download file using utility function
	if err := utils.DownloadFile(downloadURL, savePath); err != nil {
		return fmt.Errorf("file download failed [url=%s, path=%s]: %w", downloadURL, savePath, err)
	}

	return nil
}

// downloadInstallScript Fetches the official K3s installation script
func downloadInstallScript(spec *spec.K3sSpec) error {
	// Clean and construct save path for installation script
	savePath := filepath.Clean(filepath.Join(spec.DestDir, "k3s", "scripts", "k3s-install.sh"))

	// Download script using utility function
	if err := utils.DownloadFile(spec.ScriptURL, savePath); err != nil {
		return fmt.Errorf("file download failed [url=%s, path=%s]: %w", spec.ScriptURL, savePath, err)
	}

	// Set executable permission for installation script
	if err := os.Chmod(savePath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission [path=%s]: %w", savePath, err)
	}

	return nil
}
