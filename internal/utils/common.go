package utils

import (
	"compress/gzip"
	"csghub-airgap/internal/log"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
)

const (
	DefaultRetry   = 3
	DefaultTimeout = 10
)

// DownloadFile is a generic file download function
func DownloadFile(url, savePath string) error {
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Errorf("Failed to create directory %s: %v", dir, err)
		return err
	}

	var lastErr error
	for i := 0; i <= DefaultRetry; i++ {
		if i > 0 {
			log.Warnf("Download failed, retry attempt %d...", i)
			time.Sleep(time.Duration(i) * 2 * time.Second) // exponential backoff
		}

		if err := downloadOnce(url, savePath, DefaultTimeout); err != nil {
			lastErr = err
			continue
		}

		// Verify downloaded file is not empty
		fi, err := os.Stat(savePath)
		if err != nil {
			lastErr = fmt.Errorf("failed to verify file: %v", err)
			log.Error(lastErr)
			continue
		} else if fi.Size() == 0 {
			lastErr = errors.New("downloaded file is empty")
			log.Error(lastErr)
			_ = os.Remove(savePath)
			continue
		}

		log.Infof("File downloaded successfully: %s", savePath)
		return nil
	}

	return fmt.Errorf("download failed after %d retries: %v", DefaultRetry, lastErr)
}

// downloadOnce performs a single download attempt
func downloadOnce(url, savePath string, timeout int) error {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   time.Duration(timeout) * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}

	// HEAD request to get file size
	headReq, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return err
	}
	headResp, err := client.Do(headReq)
	if err != nil {
		return err
	}
	defer headResp.Body.Close()

	if headResp.StatusCode != http.StatusOK {
		return fmt.Errorf("HEAD request failed: %s", headResp.Status)
	}

	var fileSize int64 = -1
	cl := headResp.Header.Get("Content-Length")
	if cl != "" {
		size, err := strconv.ParseInt(cl, 10, 64)
		if err == nil {
			fileSize = size
		}
	}

	// Resume check
	var startPos int64 = 0
	if fi, err := os.Stat(savePath); err == nil && fi.Size() > 0 {
		startPos = fi.Size()
		if fileSize > 0 {
			if startPos >= fileSize {
				log.Infof("File already fully downloaded: %s (%d bytes)", filepath.Base(savePath), fileSize)
				return nil
			}
			log.Infof("Partial file found, resuming from: %d/%d", startPos, fileSize)
		} else {
			log.Infof("Partial file found, resuming from: %d", startPos)
		}
	}

	// GET request
	getReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if startPos > 0 {
		getReq.Header.Set("Range", fmt.Sprintf("bytes=%d-", startPos))
	}

	resp, err := client.Do(getReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("GET request failed: %s", resp.Status)
	}

	// Open file for writing
	f, err := os.OpenFile(savePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	var writer io.Writer = f
	if fileSize > 0 {
		bar := progressbar.DefaultBytes(
			fileSize,
			fmt.Sprintf("Downloading %s", filepath.Base(savePath)),
		)
		bar.Set64(startPos)
		writer = io.MultiWriter(f, bar)
	} else {
		log.Infof("Downloading %s...", filepath.Base(savePath))
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	return nil
}

// NVL An empty string fallback: Returns a fallback if the input is empty, otherwise returns the input.
func NVL(input, fallback string) string {
	if input == "" {
		return fallback
	}
	return input
}

// NVLMap An empty string fallback: Returns a fallback if the input is empty, otherwise returns the input.
func NVLMap(input, fallback map[string]interface{}) map[string]interface{} {
	if input == nil {
		return fallback
	}
	return input
}

// ExtractByRegex extracts all unique matches from the given manifest string
// based on the provided regular expression pattern. The pattern must contain
// at least one capturing group. Returns a slice of matched strings with duplicates removed.
func ExtractByRegex(manifest string, pattern string) []string {
	var results []string
	seen := make(map[string]struct{})

	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(manifest, -1)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		item := match[1]
		if _, ok := seen[item]; !ok {
			results = append(results, item)
			seen[item] = struct{}{}
		}
	}

	return results
}

// CompressToGzip compress a file into .gz format
func CompressToGzip(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	defer gz.Close()

	_, err = io.Copy(gz, in)
	return err
}

// MergeMap recursively merges src into dst.
// If both dst and src have a key with map[string]interface{} values, they are merged recursively.
// Other types are overwritten.
func MergeMap(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}
	if src == nil {
		return dst
	}

	for k, v := range src {
		if existing, ok := dst[k]; ok {
			existingMap, ok1 := existing.(map[string]interface{})
			vMap, ok2 := v.(map[string]interface{})
			if ok1 && ok2 {
				dst[k] = MergeMap(existingMap, vMap)
			} else {
				dst[k] = v
			}
		} else {
			dst[k] = v
		}
	}

	return dst
}
