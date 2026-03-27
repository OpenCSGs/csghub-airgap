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

	// ---------- Pull ----------
	pullStart := time.Now()

	imgMap := make(map[string]v1.Image)
	for _, imgName := range spec.Images {
		img, err := utils.PullImage(imgName)
		if err != nil {
			return err
		}
		imgMap[imgName] = img
	}

	if err := crane.MultiSave(imgMap, imageFilePath); err != nil {
		return fmt.Errorf("error saving images %s: %w", spec.Images, err)
	}

	log.Infof("Exported images to %s (elapsed %.3fs)", spec.ImageFileName(), time.Since(pullStart).Seconds())

	return nil
}
