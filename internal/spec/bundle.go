package spec

import "runtime"

type BundleSpec struct {
	K3s                     *K3sSpec    `yaml:"k3s"`
	SkipK3s                 bool        `yaml:"skip_k3s"`
	// Quiet suppresses interactive prompts in `deploy` (e.g. the
	// multipathd disable question) and auto-applies the safe default.
	Quiet                   bool        `yaml:"quiet,omitempty"`
	Charts                  []ChartSpec `yaml:"charts"`
	Arch                    string      `yaml:"arch"`
	Runtime                 bool        `yaml:"runtime"`
	SelectedRuntimeImages   []string    `yaml:"selected_runtime_images,omitempty"`
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
		Runtime: false,
	}, nil
}
