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
	// bundle. Ask the user whether to delete it and re-export, so they can
	// choose to keep the partial bundle instead.
	if _, err := os.Stat(incompleteFlag); err == nil {
		fmt.Printf("A previous export of %s was incomplete (some images were skipped).\n", imageFileName)
		fmt.Print("Re-export and retry the missing images? (y/N): ")
		var input string
		if _, err := fmt.Scanln(&input); err != nil || (input != "y" && input != "Y") {
			log.Warnf("Keeping the existing incomplete bundle %s", imageFileName)
			return nil
		}
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

	// Ensure the on-disk image cache dir exists before any pull writes to it.
	if err := os.MkdirAll(imageCacheDir(), 0755); err != nil {
		return err
	}

	log.Infof("Start exporting %d images to %s", len(spec.Images), imageFileName)

	pullStart := time.Now()

	// Pre-populate local containerd image list
	listLocalImages()

	localCount := 0
	remoteCount := 0
	fileCacheCount := 0

	imgMap := make(map[string]v1.Image)
	var skippedImages []string
	var tempTars []string
	var cacheHits []cacheHitEntry

	// Dump the full image list at debug so an operator can sanity-check what
	// is about to be pulled (or skipped via cache) before any network I/O.
	log.Debugf("Exporting %d image(s) for chart %s@%s:", len(spec.Images), spec.ChartName, spec.ChartVersion)
	for i, imgName := range spec.Images {
		log.Debugf("  [%d] %s", i, imgName)
	}

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
				digest := localImageDigestStr(imgName)
				localCount++
				log.Infof("[CACHE] using %s local_digest=%s (skipped remote pull)", imgName, digest)
				imgMap[imgName] = img
				cacheHits = append(cacheHits, cacheHitEntry{
					Image:       imgName,
					LocalDigest: digest,
				})
				if tempTarPath != "" {
					tempTars = append(tempTars, tempTarPath)
				}
				continue
			}
		}

		// Second-tier cache: an image pulled in a previous run (stored as a
		// per-ref tar under the user cache dir) is reused instead of hitting
		// the registry again. Falls through to a remote pull on a cache miss.
		if cached, ok, err := loadCachedImage(imgName); ok {
			digest := imgDigestStr(cached)
			fileCacheCount++
			log.Infof("[CACHE] using %s file_cache_digest=%s (skipped remote pull)", imgName, digest)
			imgMap[imgName] = cached
			cacheHits = append(cacheHits, cacheHitEntry{
				Image:       imgName,
				LocalDigest: digest,
			})
			continue
		} else if err != nil {
			return err
		}

		img, err = utils.PullImage(imgName, imageCachePath(imgName))
		if err != nil {
			log.Warnf("Failed to pull image %s, skipping it: %v", imgName, err)
			skippedImages = append(skippedImages, imgName)
			continue
		}
		digest := imgDigestStr(img)
		remoteCount++
		log.Infof("[REMOTE] pulled %s digest=%s", imgName, digest)
		imgMap[imgName] = img
	}

	// Print the cache report before any long-running MultiSave so the operator
	// can still abort if the report reveals unexpected cached images.
	printCacheReport(cacheHits)

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

	if localCount > 0 || fileCacheCount > 0 {
		log.Infof("Note: %d/%d image(s) served from local cache (containerd=%d, file cache=%d); rerun with --force-pull to refresh.",
			localCount+fileCacheCount, len(spec.Images), localCount, fileCacheCount)
	}

	log.Infof("Exported %d images to %s (elapsed %.3fs)", len(spec.Images), spec.ImageFileName(), time.Since(pullStart).Seconds())

	return nil
}

// cacheHitEntry records an image that was served from a local cache (containerd
// content store or the on-disk file cache) instead of being pulled fresh from
// the registry. Reported in the Cache Report at the end of Export.
type cacheHitEntry struct {
	Image       string
	LocalDigest string
}

// imgDigestStr returns the sha256 digest of a v1.Image as a string, or
// "<unknown>" if the digest cannot be computed.
func imgDigestStr(img v1.Image) string {
	h, err := img.Digest()
	if err != nil {
		return "<unknown>"
	}
	return h.String()
}

// printCacheReport emits a human-readable list of every image that Export
// served from the local containerd cache. No-op when there were no cache hits.
func printCacheReport(hits []cacheHitEntry) {
	if len(hits) == 0 {
		return
	}
	fmt.Println()
	log.Infof("=== Cache Report ===")
	log.Infof("%d image(s) were served from the local cache (containerd or file cache), NOT pulled fresh:", len(hits))
	log.Infof("To force a re-pull from the registry, either:")
	log.Infof("  - rerun this command with --force-pull / -fp")
	log.Infof("  - or: ctr -n k8s.io images rm <ref>  (then re-run create)")
	log.Infof("")
	nameW := 0
	for _, h := range hits {
		if len(h.Image) > nameW {
			nameW = len(h.Image)
		}
	}
	if nameW > 80 {
		nameW = 80
	}
	for _, h := range hits {
		log.Infof("  [CACHE] %-*s  local=%s", nameW, h.Image, h.LocalDigest)
	}
	fmt.Println()
}
