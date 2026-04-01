package bundle

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"encoding/json"
	"fmt"
	"sort"
)

type Spec spec.BundleSpec

func NewFromSpec(bs *spec.BundleSpec) *Spec {
	log.DebugFunc(func() string {
		data, _ := json.Marshal(bs)
		return fmt.Sprintf("Initialized BundleSpec: %s", data)
	})

	return &Spec{
		K3s:     bs.K3s,
		SkipK3s: bs.SkipK3s,
		// Sort by weight in descending order, with higher weights listed first.
		Charts: func(charts []spec.ChartSpec) []spec.ChartSpec {
			sorted := make([]spec.ChartSpec, len(charts))
			copy(sorted, charts)
			sort.SliceStable(sorted, func(i, j int) bool {
				return sorted[i].Weight > sorted[j].Weight
			})
			return sorted
		}(bs.Charts),
	}
}
