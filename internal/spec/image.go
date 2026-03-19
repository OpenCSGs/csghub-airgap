package spec

import (
	"fmt"
	"path/filepath"
)

type ImageSpec struct {
	Images       []string `yaml:"images"`
	OutPath      string   `yaml:"output"`
	ChartName    string   `yaml:"chart_name"`
	ChartVersion string   `yaml:"chart_version"`
}

func (s *ImageSpec) ImageDir() string {
	return filepath.Join(s.OutPath, s.ChartName, "images")
}

func (s *ImageSpec) ImageFileName() string {
	return fmt.Sprintf("%s-%s.tar", s.ChartName, s.ChartVersion)
}

func (s *ImageSpec) ImageFilePath() string {
	return filepath.Join(s.ImageDir(), s.ImageFileName())
}
