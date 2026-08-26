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

// Image renders the chart, extracts the images it declares, and exports
// them into the bundle. The main chart images and the user-selected runtime
// images are exported as separate tars; either can be skipped with
// skipMain / skipRuntime so `create --skip image` (or
// `--skip runtime`) can produce a partial bundle.
//
// When selectedMain is non-nil it is exported (the user picked a subset in
// the main image TUI); when nil the full extracted main image set is used.
func Image(spec *is.ChartSpec, runtimeImages []string, opts iu.ExtractOption, selectedMain []string, skipMain, skipRuntime bool) error {
	// Nothing to export for this chart — avoid even rendering.
	if skipMain && skipRuntime {
		log.Debugf("Skipping image export for chart %s (both main and runtime skipped)", spec.ChartName)
		return nil
	}

	mainImages, err := ExtractMainImages(spec, opts)
	if err != nil {
		return err
	}

	// Use the user-selected subset when provided (image-only TUI mode).
	images := mainImages
	if selectedMain != nil {
		images = selectedMain
	}

	if !skipMain {
		imageSpec := &is.ImageSpec{
			Images:       images,
			ChartName:    spec.ChartName,
			ChartVersion: spec.ChartVersion,
		}

		if err = image.Export(imageSpec); err != nil {
			return err
		}
	} else {
		log.Debugf("Skipping main image export for chart %s", spec.ChartName)
	}

	if !skipRuntime && len(runtimeImages) > 0 {
		// Registry-complete the runtime refs before export (same as before).
		for i, img := range runtimeImages {
			runtimeImages[i] = imageRegComplete(img)
		}
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

// ExtractMainImages renders the chart and returns the full set of main
// images it declares (chart images + static imageList + extra_images, all
// registry-completed). Runtime images are NOT included. Used by the
// image-only mode to build the main image selection TUI.
func ExtractMainImages(spec *is.ChartSpec, opts iu.ExtractOption) ([]string, error) {
	manifest, err := Template(spec)
	if err != nil {
		return nil, err
	}

	var yamlResource []string
	if opts.FromValues {
		values, err := ShowChartValues(spec.ChartFilePath())
		if err != nil {
			return nil, err
		}
		yamlResource = []string{string(values)}
	} else if opts.FromKind != "" {
		yamlResource, err = ShowKindOf(manifest, "KnativeServing")
		if err != nil {
			return nil, err
		}
	}

	rawImages := iu.ExtractAllImages(manifest, yamlResource, opts)
	var images []string
	for _, img := range rawImages {
		images = append(images, imageRegComplete(img))
	}

	// static images (csghub only)
	if spec.RepoName == "csghub" {
		for _, img := range imageList {
			images = append(images, imageRegComplete(img))
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

	return images, nil
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
