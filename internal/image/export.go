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
	incompleteFlag := imageFilePath + ".incomplete"

	// A previous run may have skipped some images and written an incomplete
	// bundle. Drop it so a re-run retries the missing images instead of
	// silently reusing the partial tar.
	if _, err := os.Stat(incompleteFlag); err == nil {
		log.Warnf("Previous export of %s was incomplete, re-exporting", imageFileName)
		_ = os.Remove(imageFilePath)
		_ = os.Remove(incompleteFlag)
	}

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
	var skippedImages []string
	var tempTars []string

	// Clean up temp tar files after MultiSave (or any early return) reads them.
	defer func() {
		for _, p := range tempTars {
			if err := os.Remove(p); err == nil {
				log.Debugf("Cleaned up temp tar: %s", p)
			}
		}
	}()

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
			skippedImages = append(skippedImages, imgName)
			continue
		}
		remoteCount++
		imgMap[imgName] = img
	}

	// If any images were skipped, write a partial tar with the successful
	// ones so deploy can still proceed. Mark the tar with a .incomplete flag
	// so a re-run of create will re-export and retry the missing images.
	if len(skippedImages) > 0 {
		if len(imgMap) > 0 {
			log.Warnf("Skipped %d images, writing partial bundle: %v", len(skippedImages), skippedImages)
			if err := crane.MultiSave(imgMap, imageFilePath); err != nil {
				return fmt.Errorf("error saving images %s: %v", spec.Images, err)
			}
			if err := os.WriteFile(incompleteFlag, []byte{}, 0644); err != nil {
				log.Warnf("Failed to write incomplete flag: %v", err)
			}
		} else {
			log.Warnf("No images were successfully pulled, skipping export for %s", imageFileName)
		}
		return nil
	}

	log.Infof("Exporting %d images to %s (local=%d, remote=%d), it will take a few minutes...",
		len(spec.Images), imageFileName, localCount, remoteCount)

	if err := crane.MultiSave(imgMap, imageFilePath); err != nil {
		return fmt.Errorf("error saving images %s: %v", spec.Images, err)
	}

	log.Infof("Exported %d images to %s (elapsed %.3fs)", len(spec.Images), spec.ImageFileName(), time.Since(pullStart).Seconds())

	return nil
}
