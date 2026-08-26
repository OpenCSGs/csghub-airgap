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

// Untar extracts a tar or tar.gz archive to the destination directory.
// Auto-detects if the archive is compressed with gzip.
func Untar(tarPath, destDir string) error {
	file, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %v", err)
	}
	defer file.Close()

	// Auto detect if it's a gzip file
	reader, err := autoDetectGzipReader(file)
	if err != nil {
		return fmt.Errorf("failed to detect compression: %v", err)
	}

	return untar(reader, destDir)
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

// Tar archives srcDir into a plain (uncompressed) tar at destPath. The top-level
// directory name is preserved inside the archive, so extracting restores a
// directory named after srcDir. destPath itself is excluded from the archive so
// re-packing a bundle whose archive lives next to (or inside) it stays safe.
func Tar(srcDir, destPath string) error {
	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create archive %s: %v", destPath, err)
	}

	tw := tar.NewWriter(out)

	base := filepath.Base(filepath.Clean(srcDir))
	destAbs, _ := filepath.Abs(destPath)

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Never archive the output file itself (defensive; destPath normally
		// lives in the parent of srcDir).
		if abs, _ := filepath.Abs(path); abs == destAbs {
			return nil
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.Join(base, rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		out.Close()
		return err
	}

	if err := tw.Close(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// untar extracts a tar archive from the given reader into destDir
func untar(trReader io.Reader, destDir string) error {
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
