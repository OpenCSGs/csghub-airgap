package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

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
	mode := fi.Mode()

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
