package utils

import (
	"bytes"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/utils"
	"regexp"
	"strings"
	"text/template"

	"go.yaml.in/yaml/v3"
)

type ExtractRule struct {
	Name        string              // Rule name, e.g., "csi" / "engine"
	Path        string              // Supports wildcard *, e.g., "csi.*" or "image.*"
	Template    string              // Template to construct image, e.g., "{{ if .registry }}{{ .registry }}/{{ else }}docker.io/{{ end }}{{ .repository }}:{{ .tag }}"
	ReplaceFunc func(string) string // Optional post-processing function for the extracted image string
}

type ExtractOption struct {
	FromValues bool
	FromKind   string
	Rules      []ExtractRule
}

// ExtractAllImages Merge chart images + values images, remove duplicates
func ExtractAllImages(manifest string, values []string, opt ExtractOption) []string {
	set := map[string]struct{}{}

	// 1. Images already in YAML
	for _, img := range Extract(manifest) {
		set[img] = struct{}{}
	}

	for _, value := range values {
		var valuesMap map[string]interface{}
		if err := yaml.Unmarshal([]byte(value), &valuesMap); err != nil {
			log.Fatalf("yaml parse error: %v", err)
		}

		// 2. Extract images from values.yaml
		if opt.FromValues || opt.FromKind != "" {
			imgs := ExtractImagesFromGeneric(valuesMap, opt.Rules)
			for _, img := range imgs {
				set[img] = struct{}{}
			}
		}
	}

	// 3. Remove duplicates and output
	var result []string
	for img := range set {
		result = append(result, img)
	}

	return result
}

// ExtractImagesFromGeneric Extract images from values.yaml according to rules
func ExtractImagesFromGeneric(values map[string]interface{}, rules []ExtractRule) []string {
	var result []string
	for _, rule := range rules {
		items := Walk(values, strings.Split(rule.Path, "."))
		tmpl, err := template.New("").Parse(rule.Template)
		if err != nil {
			log.Warnf("parse template error for rule %s: %v", rule.Name, err)
			continue
		}
		for _, item := range items {
			switch v := item.(type) {
			case string:
				img := strings.TrimSpace(v)
				if img == "" {
					continue
				}
				if rule.ReplaceFunc != nil {
					img = rule.ReplaceFunc(img)
				}
				result = append(result, img)
			case map[string]interface{}:
				var buf bytes.Buffer
				err = tmpl.Execute(&buf, v)
				if err != nil {
					log.Warnf("template execute error for rule %s: %v", rule.Name, err)
					continue
				}
				img := strings.TrimSpace(buf.String())
				if img == "" {
					continue
				}
				if rule.ReplaceFunc != nil {
					img = rule.ReplaceFunc(img)
				}
				result = append(result, img)
			}
		}
	}

	return result
}

// Extract all images from Manifests
func Extract(manifest string) []string {
	imgRegex1 := `image:\s*["']?([a-zA-Z0-9./:-]+(?:@[a-f0-9]+)?)["']?`
	imgRegex2 := regexp.MustCompile(`^[a-zA-Z0-9\.-]+(:[0-9]+)?/[a-zA-Z0-9_\-./]+(:[a-zA-Z0-9_\-\.]+|@sha256:[a-f0-9]{64})$`)

	imageSet := make(map[string]struct{})
	for _, img := range utils.ExtractByRegex(manifest, imgRegex1) {
		if imgRegex2.MatchString(img) {
			imageSet[img] = struct{}{}
		}
	}

	var images []string
	for img := range imageSet {
		images = append(images, img)
	}

	return images
}

// Walk traverse map/slice, supports * wildcard
func Walk(data interface{}, path []string) []interface{} {
	if len(path) == 0 {
		return []interface{}{data}
	}

	switch node := data.(type) {
	case map[string]interface{}:
		key := path[0]
		if key == "*" {
			var result []interface{}
			for _, v := range node {
				result = append(result, Walk(v, path[1:])...)
			}
			return result
		} else {
			if v, ok := node[key]; ok {
				return Walk(v, path[1:])
			}
		}
	case map[interface{}]interface{}: // May appear when unmarshaling YAML
		key := path[0]
		if key == "*" {
			var result []interface{}
			for _, v := range node {
				result = append(result, Walk(v, path[1:])...)
			}
			return result
		} else {
			if v, ok := node[key]; ok {
				return Walk(v, path[1:])
			}
		}
	case []interface{}:
		var result []interface{}
		for _, item := range node {
			result = append(result, Walk(item, path)...)
		}
		return result
	}
	return nil
}
