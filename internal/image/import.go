package image

import (
	"archive/tar"
	"context"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/distribution/reference"
)

func Import(spec *spec.ImageSpec) error {
	imageFilePath := spec.ImageFilePath()
	if _, err := os.Stat(imageFilePath); os.IsNotExist(err) {
		return fmt.Errorf("image bundle %s not found: %w\n"+
			"  The tar file does not exist on this host.\n"+
			"  If you ran create with --skip image (or --skip image is set in the bundle spec),\n"+
			"  this image set was intentionally not exported. Either:\n"+
			"    - re-run deploy without --skip image to import it, or\n"+
			"    - keep --skip image if the images are already loaded another way",
			imageFilePath, err)
	}

	if config.DryRun {
		log.Infof("[DRY-RUN] would import images from %s into containerd", imageFilePath)
		return nil
	}

	sock, err := discoverContainerdSocket()
	if err != nil {
		return err
	}
	log.Debugf("Using containerd socket: %s", sock)

	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")
	client, err := containerd.New(sock)
	if err != nil {
		return err
	}
	defer client.Close()

	// If every image in the tar already exists in containerd (compared by
	// normalized repo:tag), skip the import entirely so a repeated deploy does
	// not re-ingest the whole archive. A failure here (unreadable tar, missing
	// manifest, ...) only degrades to a regular import, never blocks deploy.
	if present, err := imagesAlreadyPresent(ctx, client, imageFilePath); err != nil {
		log.Debugf("Skipping already-imported check: %v", err)
	} else if present {
		log.Infof("All images already present, skipping import of %s", imageFilePath)
		return nil
	}

	log.Infof("Start images importing %s", imageFilePath)
	start := time.Now()

	f, err := os.Open(imageFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	images, err := client.Import(ctx, f)
	if err != nil {
		return err
	}

	log.Infof("Imported %d images into containerd (elapsed %.3fs)", len(images), time.Since(start).Seconds())
	return nil
}

// imagesAlreadyPresent reports whether all repo:tags listed in the docker-format
// image tar already exist in containerd. The comparison normalizes references
// (e.g. "foo:1.0" -> "docker.io/library/foo:1.0") so tags written without a
// registry prefix still match what containerd stored at import time.
func imagesAlreadyPresent(ctx context.Context, client *containerd.Client, imageFilePath string) (bool, error) {
	tags, err := dockerRepoTags(imageFilePath)
	if err != nil {
		return false, err
	}
	if len(tags) == 0 {
		return false, errors.New("no RepoTags found in image tar")
	}

	existing := make(map[string]struct{})
	store := client.ImageService()
	images, err := store.List(ctx)
	if err != nil {
		return false, err
	}
	for _, img := range images {
		existing[normalizeRef(img.Name)] = struct{}{}
	}

	for _, t := range tags {
		if _, ok := existing[normalizeRef(t)]; !ok {
			return false, nil
		}
	}
	return true, nil
}

// dockerRepoTags reads the manifest.json entry of a docker-format image tar and
// returns the RepoTags it declares.
func dockerRepoTags(imageFilePath string) ([]string, error) {
	f, err := os.Open(imageFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name != "manifest.json" {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		var entries []struct {
			RepoTags []string `json:"RepoTags"`
		}
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("failed to parse manifest.json: %w", err)
		}
		var tags []string
		for _, e := range entries {
			tags = append(tags, e.RepoTags...)
		}
		return tags, nil
	}
	return nil, errors.New("manifest.json not found in image tar")
}

// normalizeRef parses a docker reference and returns its canonical form with an
// explicit tag, matching how containerd stores imported image names. Falls back
// to the raw string when the reference cannot be parsed.
func normalizeRef(ref string) string {
	r, err := reference.ParseNormalizedNamed(ref)
	if err != nil {
		return ref
	}
	return reference.TagNameOnly(r).String()
}

// discoverContainerdSocket returns the first reachable containerd socket path,
// trying k3s first and falling back to the standard containerd locations.
// Used by import so it works both with a k3s host and with a stock k8s
// deployment (deploy --skip-k3s).
func discoverContainerdSocket() (string, error) {
	candidates := []string{
		"/run/k3s/containerd/containerd.sock",
		"/run/containerd/containerd.sock",
		"/var/run/containerd/containerd.sock",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no containerd socket found (tried %v)", candidates)
}
