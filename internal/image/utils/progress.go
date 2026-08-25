package utils

import (
	"csghub-airgap/internal/config"
	"io"
	"os"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/schollz/progressbar/v3"

	"csghub-airgap/internal/log"
)

var outputMu sync.Mutex

type lockedWriter struct{}

func (w lockedWriter) Write(p []byte) (n int, err error) {
	outputMu.Lock()
	defer outputMu.Unlock()
	// Always write to stdout to keep a single output stream
	return os.Stdout.Write(p)
}

func init() {
	// Ensure log package writes to the same synchronized writer
	log.SetOutput(lockedWriter{})
}

// PullImage pulls a single image, displays progress, and logs completion information
func PullImage(image string) (v1.Image, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	log.Debugf("Pulling image %s (arch=%s)", ref, config.DefaultArchitecture)
	img, err := remote.Image(
		ref,
		remote.WithPlatform(v1.Platform{
			OS:           "linux",
			Architecture: config.DefaultArchitecture},
		),
	)
	if err != nil {
		return nil, err
	}

	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}

	// Calculate total size
	var total int64
	for _, l := range layers {
		size, _ := l.Size()
		total += size
	}

	// Limit image name to max 60 chars; if longer, truncate from the left with "...".
	// If shorter, keep as-is (left-aligned).
	const maxLen = 40
	shortImage := image
	if len(shortImage) > maxLen {
		shortImage = "..." + shortImage[len(shortImage)-(maxLen-3):]
	}

	// Initialize progress bar
	bar := progressbar.NewOptions64(
		total,
		progressbar.OptionSetDescription("Pulling "+shortImage),
		progressbar.OptionSetWidth(30),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowIts(),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionSetRenderBlankState(false),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetWriter(lockedWriter{}),
	)

	// Pull layers concurrently
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup
	progressChan := make(chan int64, 100)

	done := make(chan struct{})
	go func() {
		for n := range progressChan {
			_ = bar.Add64(n)
		}
		close(done)
	}()

	for _, layer := range layers {
		layer := layer
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			rc, err := layer.Compressed()
			if err != nil {
				log.Warnf("failed to read layer %s: %v", image, err)
				return
			}
			defer rc.Close()

			buf := make([]byte, 32*1024)
			for {
				n, err := rc.Read(buf)
				if n > 0 {
					select {
					case progressChan <- int64(n):
					default:
					}
				}
				if err != nil {
					if err != io.EOF {
						log.Warnf("layer read error: %v", err)
					}
					break
				}
			}
		}()
	}

	wg.Wait()
	close(progressChan)
	<-done
	_ = bar.Finish()

	// The export loop logs a [REMOTE] ... digest=... line after PullImage
	// returns, so we keep this at Debug level to avoid duplicating the
	// success message in normal output.
	log.Debugf("Image pulled successfully: %s", image)

	return img, nil
}

// PullImages pulls multiple images sequentially with an independent progress bar for each image
func PullImages(images []string) ([]v1.Image, error) {
	var result []v1.Image
	for _, img := range images {
		imageObj, err := PullImage(img)
		if err != nil {
			return result, err
		}
		result = append(result, imageObj)
	}
	return result, nil
}
