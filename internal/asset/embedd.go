package asset

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
)

//go:embed charts/*.yaml
var ManifestsFS embed.FS

func Load() ([]string, error) {
	var contents []string

	files, err := fs.ReadDir(ManifestsFS, "charts")
	if err != nil {
		return nil, fmt.Errorf("failed to read charts directory: %w", err)
	}

	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}

		path := "charts/" + f.Name()
		data, err := fs.ReadFile(ManifestsFS, path)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		contents = append(contents, string(data))
	}

	return contents, nil
}
