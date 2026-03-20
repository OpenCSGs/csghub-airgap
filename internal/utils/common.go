package utils

import (
	"archive/tar"
	"compress/gzip"
	"csghub-airgap/internal/log"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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

// ExecCommand executes a system command with optional arguments.
// It returns the combined standard output, standard error, and any execution error.
func ExecCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)

	// Capture stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fullCmd := name + " " + strings.Join(args, " ")
		return fmt.Errorf("command failed: %s, error: %w", fullCmd, err)
	}

	return nil
}

// CopyFile copies a file from src to dst using a buffered copy.
// It creates the destination file if it does not exist, and overwrites if it does.
// Returns an error if any step fails.
func CopyFile(src, dst string, bufferSize ...int64) error {
	const defaultBufferSize = 4 * 1024 * 1024 // 4MB

	size := defaultBufferSize
	if len(bufferSize) > 0 && bufferSize[0] > 0 {
		size = int(bufferSize[0])
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Get source file permission
	fi, err := srcFile.Stat()
	if err != nil {
		return err
	}
	mode := fi.Mode() // 保存原始权限

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Open destination file with temporary permissive mode
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	// Buffered copy
	buf := make([]byte, size)
	if _, err := io.CopyBuffer(dstFile, srcFile, buf); err != nil {
		return err
	}

	// Ensure permissions are same as source
	if err := os.Chmod(dst, mode.Perm()); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", dst, err)
	}

	return nil
}

// ExtractTarGz extracts a .tar.gz archive to the specified destination directory.
// It preserves file permissions and directory structure.
func ExtractTarGz(tarGzPath, destDir string) error {
	// Open the tar.gz file
	f, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz: %w", err)
	}
	defer f.Close()

	// Create a gzip reader
	gzr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	// Create a tar reader
	tr := tar.NewReader(gzr)

	// Iterate through the files in the archive
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break // End of archive
			}
			return fmt.Errorf("error reading tar archive: %w", err)
		}

		targetPath := filepath.Join(destDir, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				return fmt.Errorf("failed to create parent dir for %s: %w", targetPath, err)
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", targetPath, err)
			}

			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file %s: %w", targetPath, err)
			}
			outFile.Close()
		default:
			// Ignore other types (symlink, block device, etc.)
		}
	}

	return nil
}

// IsZeroValue helper: check if a reflection.Value is zero
func IsZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}
