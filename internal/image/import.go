package image

import (
	"context"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	"os"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
)

func Import(spec *spec.ImageSpec) error {
	imageFilePath := spec.ImageFilePath()
	if _, err := os.Stat(imageFilePath); os.IsNotExist(err) {
		return err
	}

	log.Infof("Start images importing %s", imageFilePath)
	start := time.Now()

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
