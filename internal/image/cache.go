package image

import (
	"csghub-airgap/internal/log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// imageCacheDir returns the directory that stores pulled images as individual
// OCI tars, keyed by image reference. Re-running create (e.g. after a partial
// failure) loads these instead of re-downloading every image from the registry.
// The cache lives outside the bundle directory so it is not shipped to the
// air-gapped host.
func imageCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "csghub-airgap", "images")
}

func imageCachePath(image string) string {
	name := strings.NewReplacer("/", "_", ":", "_", "@", "_").Replace(image)
	return filepath.Join(imageCacheDir(), name+".tar")
}

// loadCachedImage returns a v1.Image for a previously pulled ref, or (nil,
// false) when no usable cache entry exists. When --force-pull is set the cache
// is bypassed entirely, matching the containerd cache behaviour.
func loadCachedImage(image string) (v1.Image, bool, error) {
	if ForcePullImages {
		return nil, false, nil
	}

	path := imageCachePath(image)
	if _, err := os.Stat(path); err != nil {
		return nil, false, nil
	}

	img, err := crane.Load(path)
	if err != nil {
		log.Warnf("Failed to load cached image %s, re-pulling: %v", image, err)
		_ = os.Remove(path)
		return nil, false, nil
	}

	// Materialize layer blobs so MultiSave can read them. The cache file stays
	// on disk, so the tar remains available for the whole export.
	layers, err := img.Layers()
	if err != nil {
		return nil, false, err
	}
	for _, layer := range layers {
		if _, err := layer.Compressed(); err != nil {
			return nil, false, err
		}
	}

	return img, true, nil
}
