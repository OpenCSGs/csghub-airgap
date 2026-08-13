package image

import (
	"csghub-airgap/internal/image/utils"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	"os"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

func Export(spec *spec.ImageSpec) error {
	imageFilePath := spec.ImageFilePath()
	imageFileName := spec.ImageFileName()

	if _, err := os.Stat(imageFilePath); err == nil {
		log.Infof("Images export files %s already exists", imageFileName)
		return nil
	}

	if err := os.MkdirAll(spec.ImageDir(), 0755); err != nil {
		return err
	}

	log.Infof("Start exporting %d images to %s", len(spec.Images), imageFileName)

	pullStart := time.Now()

	// Pre-populate local containerd image list
	listLocalImages()

	localCount := 0
	remoteCount := 0

	imgMap := make(map[string]v1.Image)
	var tempTars []string
	for _, imgName := range spec.Images {
		var img v1.Image
		var err error

		if localImageExists(imgName) {
			log.Debugf("Image found locally, exporting from containerd: %s", imgName)
			img, tempTarPath, err := exportLocalImage(imgName)
			if err != nil {
				log.Warnf("Failed to export local image %s, falling back to remote pull: %v", imgName, err)
			} else {
				localCount++
				log.Infof("Image exported from local store: %s", imgName)
				imgMap[imgName] = img
				if tempTarPath != "" {
					tempTars = append(tempTars, tempTarPath)
				}
				continue
			}
		}

		img, err = utils.PullImage(imgName)
		if err != nil {
			log.Warnf("Failed to pull image %s, skipping it: %v", imgName, err)
			continue
		}
		remoteCount++
		imgMap[imgName] = img
	}

	// Clean up temp tar files after MultiSave reads them
	defer func() {
		for _, p := range tempTars {
			if err := os.Remove(p); err == nil {
				log.Debugf("Cleaned up temp tar: %s", p)
			}
		}
	}()

	log.Infof("Exporting %d images to %s (local=%d, remote=%d), it will take a few minutes...",
		len(spec.Images), imageFileName, localCount, remoteCount)

	if err := crane.MultiSave(imgMap, imageFilePath); err != nil {
		return fmt.Errorf("error saving images %s: %v", spec.Images, err)
	}

	log.Infof("Exported %d images to %s (elapsed %.3fs)", len(spec.Images), spec.ImageFileName(), time.Since(pullStart).Seconds())

	return nil
}
