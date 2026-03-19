package chart

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
)

func Template(spec *spec.ChartSpec) (string, error) {
	settings := cli.New()

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), "", os.Getenv("HELM_DRIVER"), log.Std.Printf); err != nil {
		return "", fmt.Errorf("failed to init Helm action config: %w", err)
	}

	chatFilePath := spec.ChartFilePath()
	chartObj, err := loader.Load(chatFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart from %s: %w", chatFilePath, err)
	}

	tmpl := action.NewInstall(actionConfig)
	tmpl.DryRun = true
	tmpl.ReleaseName = "airgap"
	tmpl.Replace = true
	tmpl.ClientOnly = true
	tmpl.IncludeCRDs = true

	kubeVer, err := chartutil.ParseKubeVersion("1.25.0")
	if err != nil {
		return "", fmt.Errorf("failed to parse custom kubeVersion: %w", err)
	}
	tmpl.KubeVersion = kubeVer

	release, err := tmpl.Run(chartObj, spec.ChartOptions)
	if err != nil {
		return "", fmt.Errorf("failed to render chart: %w", err)
	}

	return release.Manifest, nil
}
