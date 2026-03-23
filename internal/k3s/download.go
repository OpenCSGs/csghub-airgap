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

	// Download Helm binary
	if err := downloadHelm(spec); err != nil {
		return fmt.Errorf("failed to download Helm binary: %w", err)
	}

	// Download K3s airgap image archive
	if err := downloadImages(spec); err != nil {
		return fmt.Errorf("failed to download K3s images: %w", err)
	}

	// Download K3s installation script
	if err := downloadInstallScript(spec); err != nil {
		return fmt.Errorf("failed to download K3s install script: %w", err)
	}

	// Download K3s patch script
	if err := downloadPatchScript(spec); err != nil {
		return fmt.Errorf("failed to download K3s patch script: %w", err)
	}

	return nil
}

// downloadHelm Downloads Helm binary for offline usage
func downloadHelm(spec *spec.K3sSpec) error {
	// Helm release package name (example: helm-v3.14.0-linux-amd64.tar.gz)
	helmFilePath := spec.K3sHelmFilePath()
	helmFileName := filepath.Base(helmFilePath)

	// Download URL
	downloadURL, _ := url.JoinPath(
		spec.HelmURL,
		helmFileName,
	)

	// Save path
	savePath := filepath.Clean(helmFilePath)

	// Download
	if err := utils.DownloadFile(downloadURL, savePath); err != nil {
		return fmt.Errorf("failed to download helm [url=%s, path=%s]: %w", downloadURL, savePath, err)
	}

	return nil
}

// downloadBinary Fetches the K3s binary executable from official release
func downloadBinary(spec *spec.K3sSpec) error {
	// Construct download URL for K3s binary
	downloadURL, _ := url.JoinPath(spec.K3sDownLoadURL(), "k3s")
	// Clean path to prevent directory traversal vulnerabilities
	savePath := spec.K3sBinFilePath()

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
	imgFileName := spec.K3sImageFileName()
	// Construct full download URL
	downloadURL, _ := url.JoinPath(spec.K3sDownLoadURL(), imgFileName)
	// Clean and construct save path
	savePath := spec.K3sImageFilePath()

	// Download file using utility function
	if err := utils.DownloadFile(downloadURL, savePath); err != nil {
		return fmt.Errorf("file download failed [url=%s, path=%s]: %w", downloadURL, savePath, err)
	}

	return nil
}

// downloadInstallScript Fetches the official K3s installation script
func downloadInstallScript(spec *spec.K3sSpec) error {
	// Clean and construct save path for installation script
	savePath := filepath.Clean(filepath.Join(spec.K3sScriptDir(), "k3s-install.sh"))

	if _, err := os.Stat(savePath); err == nil {
		return nil
	}

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

// downloadPatchScript Fetches the CoreDNS patch script
func downloadPatchScript(spec *spec.K3sSpec) error {
	// Clean and construct save path for installation script
	savePath := filepath.Clean(filepath.Join(spec.K3sScriptDir(), "k3s-coredns-patch.sh"))

	downloadURL := "https://charts.opencsg.com/repository/scripts/coredns-custom-patch.sh"
	// Download script using utility function
	if err := utils.DownloadFile(downloadURL, savePath); err != nil {
		return fmt.Errorf("file download failed [url=%s, path=%s]: %w", spec.ScriptURL, savePath, err)
	}

	// Set executable permission for installation script
	if err := os.Chmod(savePath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission [path=%s]: %w", savePath, err)
	}

	return nil
}
