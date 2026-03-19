package main

import (
	"csghub-airgap/internal/bundle"
	"csghub-airgap/internal/spec"
	"log"
)

func main() {
	spec := bundle.Spec{
		K3s:      spec.NewK3sSpec("tests"),
		Csghub:   spec.NewCsghubSpec("tests", nil),
		Longhorn: spec.NewLonghornSpec("tests", nil),
		Nvdp:     spec.NewNvdpSpec("tests", nil),
		DestDir:  "tests",
	}
	err := spec.Create()
	if err != nil {
		log.Fatal(err)
	}
}
