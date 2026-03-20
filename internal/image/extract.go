package image

import (
	"csghub-airgap/internal/utils"
	"regexp"
)

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
