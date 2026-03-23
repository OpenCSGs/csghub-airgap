package chart

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	is "csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"strings"
)

var (
	LonghornExtractOption = image.ExtractOption{
		FromValues: true,
		Rules: []image.ExtractRule{
			{
				Name:     "image.csi",
				Path:     "image.csi.*",
				Template: "{{ if .registry }}{{ .registry }}/{{ else }}docker.io/{{ end }}{{ .repository }}:{{ .tag }}",
				ReplaceFunc: func(img string) string {
					return strings.Replace(img, "docker.io", "opencsg-registry.cn-beijing.cr.aliyuncs.com/opencsghq", 1)
				},
			},
			{
				Name:     "image.longhorn",
				Path:     "image.longhorn.*",
				Template: "{{ if .registry }}{{ .registry }}/{{ else }}docker.io/{{ end }}{{ .repository }}:{{ .tag }}",
				ReplaceFunc: func(img string) string {
					return strings.Replace(img, "docker.io", "opencsg-registry.cn-beijing.cr.aliyuncs.com/opencsghq", 1)
				},
			},
		},
	}
)

func Image(spec *is.ChartSpec, opts image.ExtractOption) error {
	manifest, err := Template(spec)
	if err != nil {
		return err
	}

	valueBytes, err := ShowValuesFromTgz(spec.ChartFilePath())
	if err != nil {
		return err
	}

	images := image.ExtractAllImages(manifest, valueBytes, opts)

	imageSpec := &is.ImageSpec{
		Images:       images,
		OutPath:      utils.NVL(spec.OutPath, config.DefaultDestDir),
		ChartName:    spec.ChartName,
		ChartVersion: spec.ChartVersion,
	}

	if err = image.Export(imageSpec); err != nil {
		return err
	}

	return nil
}
