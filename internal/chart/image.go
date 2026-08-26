package chart

import (
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
		"nginx:latest",
	}
)

func Image(spec *is.ChartSpec, runtimeImages []string, opts iu.ExtractOption) error {
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

	// only csghub supports runtime images
	if spec.RepoName == "csghub" {
		// static images
		for _, img := range imageList {
			images = append(images, imageRegComplete(img))
		}

		// user-selected runtime images
		if len(runtimeImages) > 0 {
			for i, img := range runtimeImages {
				runtimeImages[i] = imageRegComplete(img)
			}
			log.Debugf("Using %d user-selected runtime images", len(runtimeImages))
		}
	}

	// extra_images declared in the chart manifest are bundled regardless of
	// repo_name; they cover cases where the chart itself does not render the
	// image (e.g. runtime-injected sidecars like argoexec).
	if len(spec.ExtraImages) > 0 {
		log.Debugf("Including %d extra images from chart manifest", len(spec.ExtraImages))
		for _, img := range spec.ExtraImages {
			images = append(images, imageRegComplete(img))
		}
	}

	imageSpec := &is.ImageSpec{
		Images:       images,
		ChartName:    spec.ChartName,
		ChartVersion: spec.ChartVersion,
	}

	if err = image.Export(imageSpec); err != nil {
		return err
	}

	if len(runtimeImages) > 0 {
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
	fullPrefix := regPrefix + "/opencsghq/"

	// Remove double registry prefix: regPrefix/opencsghq/regPrefix/... -> regPrefix/opencsghq/...
	if strings.HasPrefix(img, fullPrefix+regPrefix+"/") {
		img = fullPrefix + strings.TrimPrefix(img, fullPrefix+regPrefix+"/")
	}

	// Remove double namespace prefix: regPrefix/opencsghq/opencsghq/... -> regPrefix/opencsghq/...
	if strings.HasPrefix(img, fullPrefix+"opencsghq/") {
		img = fullPrefix + strings.TrimPrefix(img, fullPrefix+"opencsghq/")
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
