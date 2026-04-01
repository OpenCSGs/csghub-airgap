package image

import (
	"context"
	"csghub-airgap/internal/log"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

type GitHubTree struct {
	Tree []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

func Runtime(ctx context.Context) ([]string, error) {
	apiURL := "https://api.github.com/repos/OpenCSGs/csghub-server/git/trees/main?recursive=1"

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch repo tree: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching tree: %d", resp.StatusCode)
	}

	var result GitHubTree
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("cannot parse repo tree: %w", err)
	}

	var images []string

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	var mu sync.Mutex

	for _, f := range result.Tree {
		f := f

		if f.Type != "blob" ||
			!strings.HasPrefix(f.Path, "configs/") ||
			!strings.HasSuffix(f.Path, ".json") {
			continue
		}

		g.Go(func() error {
			rawURL := fmt.Sprintf(
				"https://ghfast.top/https://raw.githubusercontent.com/OpenCSGs/csghub-server/main/%s",
				f.Path,
			)

			data, err := downloadToMemory(ctx, rawURL)
			if err != nil {
				log.Warnf("Cannot fetch %s contents: %v", f.Path, err)
				return nil
			}

			var obj struct {
				EngineImages []struct {
					Image string `json:"image"`
				} `json:"engine_images"`
			}

			if err := json.Unmarshal(data, &obj); err != nil {
				log.Warnf("Invalid JSON %s: %v", f.Path, err)
				return nil
			}

			mu.Lock()
			for _, img := range obj.EngineImages {
				if img.Image != "" {
					images = append(images, img.Image)
				}
			}
			mu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return images, nil
}

func downloadToMemory(ctx context.Context, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status downloading file: %d", resp.StatusCode)
	}

	// limit to 10MB to avoid excessive memory usage
	const maxSize = 10 << 20
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxSize))
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	return data, nil
}
