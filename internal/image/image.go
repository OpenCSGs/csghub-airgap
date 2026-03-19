package image

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"os"
	"regexp"
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
		img, err := crane.Pull(imgName)
		if err != nil {
			return err
		}
		imgMap[imgName] = img
	}

	if err := crane.MultiSave(imgMap, imageFilePath); err != nil {
		return err
	}

	log.Infof("Exported images to %s (elapsed %.3fs)", spec.ImageFileName(), time.Since(pullStart).Seconds())

	return nil
}

func Import(spec *spec.ImageSpec) error {
	return nil
}

// Extract all images from Manifests
func Extract(manifest string) []string {
	imgRegex1 := `(?m)^[^#]*\s*image:\s*"?([a-z0-9_\-./:@]+(@sha256:[a-f0-9]+)?)"?\s*$`
	imgRegex2 := regexp.MustCompile(`([a-z0-9\.-]+(:[0-9]+)?/)?[a-z0-9_\-./]+:[a-z0-9_\-\.]+`)

	imageSet := make(map[string]struct{})
	for _, img := range utils.ExtractByRegex(manifest, imgRegex1) {
		if imgRegex2.MatchString(img) {
			imageSet[img] = struct{}{}
		}
	}

	var images []string
	for img := range imageSet {
		images = append(images, img)
	}

	return images
}
