package spec

import "runtime"

type BundleSpec struct {
	K3s                     *K3sSpec    `yaml:"k3s"`
	SkipK3s                 bool        `yaml:"skip_k3s"`
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

	return &BundleSpec{
		K3s:     NewK3sSpec(),
		SkipK3s: false,
		Charts:  charts,
		Arch:    runtime.GOARCH,
		Runtime: false,
	}, nil
}
