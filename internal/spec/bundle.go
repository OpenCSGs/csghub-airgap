package spec

type BundleSpec struct {
	K3s     *K3sSpec    `yaml:"k3s"`
	SkipK3s bool        `yaml:"skip_k3s"`
	Charts  []ChartSpec `yaml:"charts"`
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
	}, nil
}
