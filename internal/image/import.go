package image

import (
	"context"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
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

	ctx := namespaces.WithNamespace(context.Background(), "k8s.io")
	client, err := containerd.New("/run/k3s/containerd/containerd.sock")
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

	log.Infof("Imported %d images into k3s (elapsed %.3fs)", len(images), time.Since(start).Seconds())
	return nil
}

//// Import imports images from a multi-image tar file and pushes them to a remote registry
//func Import(spec *spec.ImageSpec) error {
//	imageFilePath := spec.ImageFilePath()
//	imageFileName := spec.ImageFileName()
//
//	if _, err := os.Stat(imageFilePath); os.IsNotExist(err) {
//		return err
//	}
//
//	log.Infof("Start importing images from %s", imageFileName)
//	loadStart := time.Now()
//
//	// Load the manifest from the tar file
//	manifest, err := loadManifestFromFile(imageFilePath)
//	if err != nil {
//		return err
//	}
//
//	// Iterate over each image in the manifest
//	for _, m := range manifest {
//		for _, repoTag := range m.RepoTags {
//			log.Infof("Importing image: %s", repoTag)
//
//			// Parse the image tag
//			tag, err := name.NewTag(repoTag)
//			if err != nil {
//				log.Errorf("Invalid tag %s: %v", repoTag, err)
//				return err
//			}
//
//			// Load a single image from the tar
//			img, err := tarball.Image(func() (io.ReadCloser, error) {
//				return os.Open(imageFilePath)
//			}, &tag)
//			if err != nil {
//				log.Errorf("Failed to load image %s: %v", repoTag, err)
//				return err
//			}
//
//			// Push the image to the remote registry
//			if err := crane.Push(img, repoTag); err != nil {
//				log.Errorf("Failed to push image %s: %v", repoTag, err)
//				return err
//			}
//		}
//	}
//
//	log.Infof("Imported images from %s (elapsed %.3fs)", imageFileName, time.Since(loadStart).Seconds())
//	return nil
//}
//
//// loadManifestFromFile loads the manifest from a multi-image tar file using tarball.LoadManifest
//func loadManifestFromFile(tarPath string) (tarball.Manifest, error) {
//	return tarball.LoadManifest(func() (io.ReadCloser, error) {
//		return os.Open(tarPath)
//	})
//}
