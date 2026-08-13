package extract

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// UnTarGz extracts a tar or tar.gz archive to the destination directory.
// Auto-detects if the archive is compressed with gzip.
func UnTarGz(tarGzPath, destDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer file.Close()

	// Auto detect if it's a gzip file
	reader, err := autoDetectGzipReader(file)
	if err != nil {
		return fmt.Errorf("failed to detect compression: %v", err)
	}

	return UnTar(reader, destDir)
}

// autoDetectGzipReader returns an appropriate reader:
// if file is gzip -> returns gzip reader
// else -> returns raw file reader
func autoDetectGzipReader(file *os.File) (io.Reader, error) {
	gr, err := gzip.NewReader(file)
	if err == nil {
		return gr, nil
	}

	// Not gzip: reset file pointer and use raw reader
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file: %v", err)
	}

	return file, nil
}

// UnTar extracts a tar archive from the given reader into destDir
func UnTar(trReader io.Reader, destDir string) error {
	tr := tar.NewReader(trReader)

	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to read tar entry: %v", err)
		}

		targetPath := filepath.Join(destDir, hdr.Name)

		// Prevent path traversal: the resolved path must stay inside destDir.
		rel, err := filepath.Rel(destDir, targetPath)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("tar entry %q escapes destination directory %s", hdr.Name, destDir)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(hdr.Mode)); err != nil {
				return fmt.Errorf("failed to create dir %s: %v", targetPath, err)
			}

		case tar.TypeReg:
			parentDir := filepath.Dir(targetPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				return fmt.Errorf("failed to create parent dir %s: %v", parentDir, err)
			}

			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %v", targetPath, err)
			}

			_, copyErr := io.Copy(outFile, tr)
			closeErr := outFile.Close()

			if copyErr != nil {
				return fmt.Errorf("failed to write file %s: %v", targetPath, copyErr)
			}
			if closeErr != nil {
				return fmt.Errorf("failed to close file %s: %v", targetPath, closeErr)
			}
		}
	}

	return nil
}
