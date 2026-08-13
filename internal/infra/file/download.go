package file

import (
	"csghub-airgap/internal/log"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
		return fmt.Errorf("failed to create directory %s: %v", dir, err)
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
			log.Infof("File exists and Content-Length not provided, skipping download: %s", filepath.Base(savePath))
			return nil
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

	// If a Range was requested but the server ignored it (returned 200 with the
	// full body), appending would duplicate data. Fall back to a full re-download.
	if startPos > 0 && resp.StatusCode != http.StatusPartialContent {
		log.Warnf("Server did not honor Range request, restarting download of %s", filepath.Base(savePath))
		startPos = 0
	}

	// Open file for writing
	flags := os.O_CREATE | os.O_WRONLY
	if startPos > 0 {
		flags |= os.O_APPEND
	}
	f, err := os.OpenFile(savePath, flags, 0755)
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
