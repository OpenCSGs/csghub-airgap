package bundle

import (
	"csghub-airgap/internal/spec"
)

type Spec spec.BundleSpec

func NewFromSpec(bs *spec.BundleSpec) *Spec {
	return &Spec{
		K3s:          bs.K3s,
		SkipK3s:      bs.SkipK3s,
		Csghub:       bs.Csghub,
		Longhorn:     bs.Longhorn,
		SkipLonghorn: bs.SkipLonghorn,
		Nvdp:         bs.Nvdp,
		SkipNvdp:     bs.SkipNvdp,
		Arch:         bs.Arch,
		DestDir:      bs.DestDir,
	}
}
