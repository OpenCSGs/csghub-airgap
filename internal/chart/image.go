package chart

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/log"
	is "csghub-airgap/internal/spec"
	"csghub-airgap/internal/utils"
	"fmt"
	"strings"
)

var (
	LonghornExtractOption = image.ExtractOption{
		FromValues: true,
		FromKind:   "",
		Rules: []image.ExtractRule{
			{
				Name:     "image.csi",
				Path:     "image.csi.*",
				Template: "{{ if .registry }}{{ .registry }}/{{ else }}docker.io/{{ end }}{{ .repository }}:{{ .tag }}",
				ReplaceFunc: func(img string) string {
					return strings.Replace(img, "docker.io", config.DefaultRegistryWithNamespace, 1)
				},
			},
			{
				Name:     "image.longhorn",
				Path:     "image.longhorn.*",
				Template: "{{ if .registry }}{{ .registry }}/{{ else }}docker.io/{{ end }}{{ .repository }}:{{ .tag }}",
				ReplaceFunc: func(img string) string {
					return strings.Replace(img, "docker.io", config.DefaultRegistryWithNamespace, 1)
				},
			},
		},
	}
	CsghubExtractOption = image.ExtractOption{
		FromValues: false,
		FromKind:   "KnativeServing",
		Rules: []image.ExtractRule{
			{
				Name:     "spec.registry.override",
				Path:     "spec.registry.override.*",
				Template: "{{ . }}",
				ReplaceFunc: func(img string) string {
					return strings.Replace(img, "gcr.io", config.DefaultRegistryWithNamespace, 1)
				},
			},
		},
	}
	imageList = []string{
		"space-base:python3.10-1.0.4",
		"space-base:python3.10-cuda12.1.0-1.0.4",
		"space-base:python3.10-cuda11.8.0-1.0.4",
		"nginx:latest",
		"argoproj/argoexec:v3.7.10",
	}
)

func Image(spec *is.ChartSpec, opts image.ExtractOption) error {
	manifest, err := Template(spec)
	if err != nil {
		return err
	}

	var yamlResource []string
	if opts.FromValues {
		values, err := ShowChartValues(spec.ChartFilePath())
		if err != nil {
			return err
		}
		yamlResource = []string{string(values)}
	} else if opts.FromKind != "" {
		yamlResource, err = ShowKindOf(manifest, "KnativeServing")
		if err != nil {
			return err
		}
	}

	images := image.ExtractAllImages(manifest, yamlResource, opts)

	switch spec.RepoName {
	case "csghub":
		for _, img := range imageList {
			images = append(images, fmt.Sprintf("%s/%s", config.DefaultRegistryWithNamespace, img))
		}
	}

	log.Debugf("Images pulled: %v", images)

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
