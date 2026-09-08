package spec

import "runtime"

type BundleSpec struct {
	K3s     *K3sSpec `yaml:"k3s"`
	SkipK3s bool     `yaml:"skip_k3s"`
	// Quiet suppresses interactive prompts in `deploy` (e.g. the
	// multipathd disable question) and auto-applies the safe default.
	Quiet bool `yaml:"quiet,omitempty"`
	// SkipCharts skips pulling (create) / installing (deploy) the Helm
	// charts, reusing chart files that already exist. Main images and
	// runtime images are still exported/imported.
	SkipCharts bool `yaml:"skip_charts,omitempty"`
	// SkipImages skips exporting (create) / importing (deploy) the main
	// chart images only. Runtime images are independent — see SkipRuntimeImages.
	SkipImages bool `yaml:"skip_images,omitempty"`
	// SkipRuntimeImages skips exporting (create) / importing (deploy) the
	// user-selected runtime images. Charts and main images still run.
	SkipRuntimeImages bool `yaml:"skip_runtime_images,omitempty"`
	// SkipPkg skips the NVIDIA container toolkit package download (create)
	// and install (deploy), independent of the nvdp chart state. It bypasses
	// the package download, its cache validation, and the install step.
	SkipPkg bool `yaml:"skip_pkg,omitempty"`
	// SkipTar skips packing the bundle directory into a tar (create) and
	// unpacking it (deploy). When set, create leaves the bundle as a plain
	// directory and deploy expects an already-unpacked directory.
	SkipTar               bool        `yaml:"skip_tar,omitempty"`
	Charts                []ChartSpec `yaml:"charts"`
	Arch                  string      `yaml:"arch"`
	SelectedRuntimeImages []string    `yaml:"selected_runtime_images,omitempty"`
}

func LoadBundleSpec() (*BundleSpec, error) {
	charts, err := NewChartSpec()
	if err != nil {
		return nil, err
	}

	ks := NewK3sSpec()
	// Bundles are only ever deployed on Linux, so always bundle the Linux
	// helm binary even when create runs on another OS (e.g. macOS).
	ks.TargetOS = "linux"

	return &BundleSpec{
		K3s:     ks,
		SkipK3s: false,
		Charts:  charts,
		Arch:    runtime.GOARCH,
	}, nil
}
