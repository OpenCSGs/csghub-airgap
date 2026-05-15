package chart

import (
	"context"
	"csghub-airgap/internal/image"
	iu "csghub-airgap/internal/image/utils"
	"csghub-airgap/internal/log"
	is "csghub-airgap/internal/spec"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	LonghornExtractOption = iu.ExtractOption{
		FromValues: true,
		FromKind:   "",
		Rules: []iu.ExtractRule{
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
	CsghubExtractOption = iu.ExtractOption{
		FromValues: false,
		FromKind:   "KnativeServing",
		Rules: []iu.ExtractRule{
			{
				Name:     "spec.registry.override",
				Path:     "spec.registry.override.*",
				Template: "{{ . }}",
				ReplaceFunc: func(img string) string {
					return strings.Replace(img, "gcr.io", "opencsg-registry.cn-beijing.cr.aliyuncs.com/opencsghq", 1)
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

func Image(spec *is.ChartSpec, runtime bool, opts iu.ExtractOption) error {
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

	rawImages := iu.ExtractAllImages(manifest, yamlResource, opts)
	var images []string
	for _, img := range rawImages {
		images = append(images, imageRegComplete(img))
	}
	var runtimeImages []string

	// only csghub supports runtime images
	if spec.RepoName == "csghub" {
		// static images
		for _, img := range imageList {
			images = append(images, imageRegComplete(img))
		}

		// dynamic runtime images
		if runtime {
			ctx := context.Background()
			runtimeImages, err = image.Runtime(ctx)
			if err != nil {
				return err
			}

			if len(runtimeImages) == 0 {
				log.Warn("runtime enabled but no runtime images found")
			} else {
				for i, img := range runtimeImages {
					runtimeImages[i] = imageRegComplete(img)
				}
			}
		}

		log.Debugf("Images will pulling: %v, runtime: %v", images, runtimeImages)
	} else {
		log.Debugf("Images will pulling: %v", images)
	}

	imageSpec := &is.ImageSpec{
		Images:       images,
		ChartName:    spec.ChartName,
		ChartVersion: spec.ChartVersion,
	}

	if err = image.Export(imageSpec); err != nil {
		return err
	}

	if runtime && len(runtimeImages) > 0 {
		imageSpec := &is.ImageSpec{
			Images:       runtimeImages,
			ChartName:    fmt.Sprintf("%s-%s", spec.ChartName, "runtime"),
			ChartVersion: spec.ChartVersion,
		}

		if err = image.Export(imageSpec); err != nil {
			return err
		}
	}

	return nil
}

func imageRegComplete(img string) string {
	regPrefix := "opencsg-registry.cn-beijing.cr.aliyuncs.com"

	// Remove double registry prefix: regPrefix/opencsghq/regPrefix/... -> regPrefix/opencsghq/...
	doublePrefix := regPrefix + "/opencsghq/" + regPrefix + "/"
	if strings.HasPrefix(img, doublePrefix) {
		img = regPrefix + "/opencsghq/" + strings.TrimPrefix(img, doublePrefix)
	}

	parts := strings.Split(img, "/")

	// already has registry (first part contains '.' or ':')
	if len(parts) > 1 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		return img
	}

	if !strings.HasPrefix(img, "opencsghq/") {
		return filepath.Join(regPrefix, "opencsghq", img)
	}

	return filepath.Join(regPrefix, img)
}
