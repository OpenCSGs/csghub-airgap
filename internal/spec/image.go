package spec

import (
	"csghub-airgap/internal/config"
	"fmt"
	"path/filepath"
)

type ImageSpec struct {
	Images       []string `yaml:"images"`
	ChartName    string   `yaml:"chart_name"`
	ChartVersion string   `yaml:"chart_version"`
}

func (s *ImageSpec) ImageDir() string {
	return filepath.Join(config.DefaultBaseDir, s.ChartName, "images")
}

func (s *ImageSpec) ImageFileName() string {
	return fmt.Sprintf("%s-%s.tar", s.ChartName, s.ChartVersion)
}

func (s *ImageSpec) ImageFilePath() string {
	return filepath.Join(s.ImageDir(), s.ImageFileName())
}
