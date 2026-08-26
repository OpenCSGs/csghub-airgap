package utils

import (
	"csghub-airgap/internal/config"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
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

// countingRoundTripper wraps the default HTTP transport so blob bytes streamed
// from the registry are counted towards the progress bar as they are read.
type countingRoundTripper struct {
	base http.RoundTripper
	bar  *progressbar.ProgressBar
}

func (c *countingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := c.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}
	if resp.Body != nil {
		resp.Body = &countingBody{ReadCloser: resp.Body, bar: c.bar}
	}
	return resp, nil
}

type countingBody struct {
	io.ReadCloser
	bar *progressbar.ProgressBar
}

func (b *countingBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		_ = b.bar.Add64(int64(n))
	}
	return n, err
}

// PullImage downloads a single image exactly once and persists it to a tar at
// dest. The returned v1.Image is backed by that tar (loaded from disk), so
// subsequent reads (e.g. MultiSave) never hit the registry again. dest doubles
// as a cache: a later run that already has the tar can skip the download.
func PullImage(image string, dest string) (v1.Image, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, err
	}

	log.Debugf("Pulling image %s (arch=%s)", ref, config.DefaultArchitecture)

	// Limit image name to max 40 chars; if longer, truncate from the left with "...".
	const maxLen = 40
	shortImage := image
	if len(shortImage) > maxLen {
		shortImage = "..." + shortImage[len(shortImage)-(maxLen-3):]
	}

	bar := progressbar.NewOptions64(
		1,
		progressbar.OptionSetDescription("Pulling "+shortImage),
		progressbar.OptionSetWidth(30),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowIts(),
		progressbar.OptionThrottle(100*time.Millisecond),
		progressbar.OptionSetRenderBlankState(false),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetWriter(lockedWriter{}),
	)

	// Count every byte pulled from the registry towards the progress bar.
	rt := &countingRoundTripper{base: http.DefaultTransport, bar: bar}
	img, err := remote.Image(
		ref,
		remote.WithPlatform(v1.Platform{
			OS:           "linux",
			Architecture: config.DefaultArchitecture,
		}),
		remote.WithTransport(rt),
	)
	if err != nil {
		return nil, err
	}

	// The bar needs the total layer size, which requires layer metadata (HEAD
	// requests) — cheap compared to the blob download that follows.
	layers, err := img.Layers()
	if err != nil {
		return nil, err
	}
	var total int64
	for _, l := range layers {
		size, _ := l.Size()
		total += size
	}
	if total > 0 {
		bar.ChangeMax64(total)
	}

	// Single download: write the image straight into dest (atomically via a
	// .tmp sibling), then serve subsequent reads from disk.
	tmp := dest + ".tmp"
	if err := crane.Save(img, image, tmp); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return nil, err
	}
	_ = bar.Finish()

	log.Debugf("Image pulled successfully: %s", image)
	return crane.Load(dest)
}

// PullImages pulls multiple images sequentially, writing each to dest[i].
func PullImages(images, dests []string) ([]v1.Image, error) {
	var result []v1.Image
	for i, img := range images {
		var dest string
		if i < len(dests) {
			dest = dests[i]
		}
		imageObj, err := PullImage(img, dest)
		if err != nil {
			return result, err
		}
		result = append(result, imageObj)
	}
	return result, nil
}
