package image

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

var (
	cachedLocalImages map[string]struct{}
	localOnce         sync.Once
)

// ctrCommand builds the exec.Cmd for a ctr operation.
// If k3s is on PATH, it uses "k3s ctr -n k8s.io ..."; otherwise "ctr -n k8s.io ...".
func ctrCommand(args ...string) *exec.Cmd {
	if _, err := exec.LookPath("k3s"); err == nil {
		fullArgs := append([]string{"ctr", "-n", "k8s.io"}, args...)
		return exec.Command("k3s", fullArgs...)
	}
	fullArgs := append([]string{"-n", "k8s.io"}, args...)
	return exec.Command("ctr", fullArgs...)
}

// listLocalImages returns a set of image references present in the local containerd store.
func listLocalImages() map[string]struct{} {
	localOnce.Do(func() {
		cachedLocalImages = make(map[string]struct{})

		cmd := ctrCommand("images", "ls", "-q")
		out, err := cmd.Output()
		if err != nil {
			log.Debugf("Failed to list local containerd images: %v", err)
			return
		}

		for _, line := range strings.Split(string(out), "\n") {
			ref := strings.TrimSpace(line)
			if ref != "" {
				cachedLocalImages[ref] = struct{}{}
			}
		}
		log.Debugf("Found %d images in local containerd store", len(cachedLocalImages))
	})
	return cachedLocalImages
}

// localImageExists checks whether the given image reference exists in the local containerd store.
func localImageExists(image string) bool {
	local := listLocalImages()
	if len(local) == 0 {
		return false
	}

	// exact match
	if _, ok := local[image]; ok {
		return true
	}

	// Some registries add library/ prefix for dockerhub images
	if _, ok := local["docker.io/library/"+image]; ok {
		return true
	}

	return false
}

// exportLocalImage exports an image from the local containerd store and returns it as a v1.Image.
// The temporary tar file is fully consumed (layer blobs materialized) before the function returns,
// so no external cleanup is needed.
func exportLocalImage(image string) (v1.Image, error) {
	tmpFile, err := os.CreateTemp("", "ctr-export-*.tar")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	arch := config.DefaultArchitecture
	cmd := ctrCommand("images", "export", "--platform", "linux/"+arch, tmpPath, image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("ctr images export failed: %w, output: %s", err, string(output))
	}

	img, err := crane.Load(tmpPath)
	if err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("crane load failed for %s: %w", tmpPath, err)
	}

	// Materialize all layer blobs into memory so the temp tar can be deleted immediately.
	// This avoids leaving large temp files visible during the subsequent MultiSave.
	layers, err := img.Layers()
	if err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("failed to read layers for %s: %w", image, err)
	}
	for _, layer := range layers {
		if _, err := layer.Compressed(); err != nil {
			os.Remove(tmpPath)
			return nil, fmt.Errorf("failed to materialize layer for %s: %w", image, err)
		}
	}

	os.Remove(tmpPath)
	return img, nil
}
