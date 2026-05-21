package image

import (
	"context"
	"csghub-airgap/internal/log"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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

// RuntimeImageInfo represents a single runtime image with its metadata.
type RuntimeImageInfo struct {
	Category    string `json:"category"`     // e.g. "finetune", "inference", "evaluation"
	EngineName  string `json:"engine_name"`  // e.g. "llama-factory", "vllm"
	ComputeType string `json:"compute_type"` // e.g. "cpu", "gpu", "dcu"
	Image       string `json:"image"`
	EngineVer   string `json:"engine_version"`
	DriverVer   string `json:"driver_version"`
}

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

// Source specifies where to fetch runtime image configs from.
type Source string

const (
	SourceGitHub Source = "github"
	SourceLocal  Source = "local"
)

// ListRuntimeImages fetches runtime image configs and returns a deduplicated list of
// RuntimeImageInfo. For SourceGitHub, version is a branch name (main, dev) or tag (v1.2.0).
// For SourceLocal, version is the local directory path containing the configs/ folder.
// categoryFilter and computeFilter are optional — empty means all.
func ListRuntimeImages(ctx context.Context, source Source, version string, categoryFilter, computeFilter []string) ([]RuntimeImageInfo, error) {
	switch source {
	case SourceLocal:
		return listRuntimeImagesLocal(ctx, version, categoryFilter, computeFilter)
	case SourceGitHub:
		return listRuntimeImagesGitHub(ctx, version, categoryFilter, computeFilter)
	default:
		return nil, fmt.Errorf("unknown source: %s (supported: github, local)", source)
	}
}

func listRuntimeImagesGitHub(ctx context.Context, version string, categoryFilter, computeFilter []string) ([]RuntimeImageInfo, error) {
	ref := version
	if ref == "" {
		ref = "main"
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/OpenCSGs/csghub-server/git/trees/%s?recursive=1", ref)

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

	var (
		allImages []RuntimeImageInfo
		mu        sync.Mutex
	)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	for _, f := range result.Tree {
		f := f

		if f.Type != "blob" ||
			!strings.HasPrefix(f.Path, "configs/") ||
			!strings.HasSuffix(f.Path, ".json") {
			continue
		}

		category := extractCategory(f.Path)
		if category == "" {
			continue
		}

		// Filter by category
		if len(categoryFilter) > 0 && !slices.Contains(categoryFilter, category) {
			continue
		}

		g.Go(func() error {
			rawURL := fmt.Sprintf(
				"https://ghfast.top/https://raw.githubusercontent.com/OpenCSGs/csghub-server/%s/%s",
				ref, f.Path,
			)

			data, err := downloadToMemory(ctx, rawURL)
			if err != nil {
				log.Warnf("Cannot fetch %s contents: %v", f.Path, err)
				return nil
			}

			imgs := parseConfigData(category, data, computeFilter)
			if len(imgs) > 0 {
				mu.Lock()
				allImages = append(allImages, imgs...)
				mu.Unlock()
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return dedupImages(allImages), nil
}

func listRuntimeImagesLocal(ctx context.Context, configDir string, categoryFilter, computeFilter []string) ([]RuntimeImageInfo, error) {
	// configDir should contain a configs/ subdirectory, or be the configs dir itself
	configsDir := configDir
	if base := filepath.Base(configDir); base != "configs" {
		configsDir = filepath.Join(configDir, "configs")
	}

	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return nil, fmt.Errorf("cannot read configs directory %s: %w", configsDir, err)
	}

	var allImages []RuntimeImageInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		category := entry.Name()
		if len(categoryFilter) > 0 && !slices.Contains(categoryFilter, category) {
			continue
		}

		catDir := filepath.Join(configsDir, category)
		files, err := os.ReadDir(catDir)
		if err != nil {
			log.Warnf("Cannot read %s: %v", catDir, err)
			continue
		}

		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}

			data, err := os.ReadFile(filepath.Join(catDir, f.Name()))
			if err != nil {
				log.Warnf("Cannot read %s: %v", f.Name(), err)
				continue
			}

			imgs := parseConfigData(category, data, computeFilter)
			allImages = append(allImages, imgs...)
		}
	}

	return dedupImages(allImages), nil
}

// extractCategory gets the first path component after "configs/".
func extractCategory(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "configs/"), "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

func parseConfigData(category string, data []byte, computeFilter []string) []RuntimeImageInfo {
	var obj struct {
		EngineName   string `json:"engine_name"`
		EngineImages []struct {
			Image       string `json:"image"`
			ComputeType string `json:"compute_type"`
			EngineVer   string `json:"engine_version"`
			DriverVer   string `json:"driver_version"`
		} `json:"engine_images"`
	}

	if err := json.Unmarshal(data, &obj); err != nil {
		log.Warnf("Invalid JSON for category %s: %v", category, err)
		return nil
	}

	var result []RuntimeImageInfo
	for _, ei := range obj.EngineImages {
		if ei.Image == "" {
			continue
		}
		if len(computeFilter) > 0 && !slices.Contains(computeFilter, ei.ComputeType) {
			continue
		}
		result = append(result, RuntimeImageInfo{
			Category:    category,
			EngineName:  obj.EngineName,
			ComputeType: ei.ComputeType,
			Image:       ei.Image,
			EngineVer:   ei.EngineVer,
			DriverVer:   ei.DriverVer,
		})
	}
	return result
}

func dedupImages(images []RuntimeImageInfo) []RuntimeImageInfo {
	seen := make(map[string]bool)
	var deduped []RuntimeImageInfo
	for _, img := range images {
		if !seen[img.Image] {
			seen[img.Image] = true
			deduped = append(deduped, img)
		}
	}
	return deduped
}

// Runtime fetches runtime images from the main branch for backward compatibility.
// Deprecated: use ListRuntimeImages instead.
func Runtime(ctx context.Context) ([]string, error) {
	images, err := ListRuntimeImages(ctx, SourceGitHub, "main", nil, nil)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, img := range images {
		result = append(result, img.Image)
	}
	return result, nil
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
