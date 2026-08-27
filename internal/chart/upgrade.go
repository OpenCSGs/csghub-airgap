package chart

import (
	"context"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	logger "log"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
)

func Upgrade(spec *spec.ChartSpec) error {
	if config.DryRun {
		log.Infof("[DRY-RUN] $ %s", HelmUpgradeCmd(spec))
		return nil
	}

	ctx := context.Background()

	namespace := spec.RepoName

	// Initialize Helm settings
	settings := cli.New()
	kubeconfig, err := DiscoverKubeConfig()
	if err != nil {
		return err
	}
	settings.KubeConfig = kubeconfig
	log.Debugf("Using kubeconfig: %s", kubeconfig)

	if namespace != "" {
		settings.SetNamespace(namespace)
	}

	chartRef := spec.ChartFilePath()
	chartVersion := spec.ChartVersion
	releaseName := spec.RepoName

	if releaseName == "" {
		return fmt.Errorf("release name must be specified")
	}

	log.Infof("Upgrading release %s with chart %s", releaseName, chartRef)

	// Values (can be empty)
	values := map[string]interface{}{}
	if spec.Values != nil {
		values = spec.Values
	}

	// Run upgrade
	if err := runUpgrade(ctx, log.Std, settings, releaseName, chartRef, chartVersion, values); err != nil {
		return err
	}

	log.Infof("Upgrade completed: %s", releaseName)

	return nil
}

func runUpgrade(ctx context.Context, logger *logger.Logger, settings *cli.EnvSettings, releaseName string, chartRef string, chartVersion string, releaseValues map[string]interface{}) error {
	actionConfig, err := initActionConfig(settings, logger)
	if err != nil {
		return fmt.Errorf("failed to init action config: %v", err)
	}

	upgradeClient := action.NewUpgrade(actionConfig)

	upgradeClient.Namespace = settings.Namespace()
	upgradeClient.DryRunOption = "none"
	upgradeClient.Version = chartVersion
	upgradeClient.Install = true

	registryClient, err := newRegistryClientTLS(
		settings,
		logger,
		upgradeClient.CertFile,
		upgradeClient.KeyFile,
		upgradeClient.CaFile,
		upgradeClient.InsecureSkipTLSverify,
		upgradeClient.PlainHTTP)
	if err != nil {
		return fmt.Errorf("missing registry client: %v", err)
	}

	upgradeClient.SetRegistryClient(registryClient)

	chartPath, err := upgradeClient.ChartPathOptions.LocateChart(chartRef, settings)
	if err != nil {
		return err
	}

	providers := getter.All(settings)

	// Check chart dependencies to make sure all are present in /charts
	chart, err := loader.Load(chartPath)
	if err != nil {
		return fmt.Errorf("failed to load chart: %v", err)
	}
	if req := chart.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chart, req); err != nil {
			err = fmt.Errorf("failed to check chart dependencies: %v", err)
			if !upgradeClient.DependencyUpdate {
				return err
			}

			man := &downloader.Manager{
				Out:              logger.Writer(),
				ChartPath:        chartPath,
				Keyring:          upgradeClient.ChartPathOptions.Keyring,
				SkipUpdate:       false,
				Getters:          providers,
				RepositoryConfig: settings.RepositoryConfig,
				RepositoryCache:  settings.RepositoryCache,
				Debug:            settings.Debug,
			}
			if err := man.Update(); err != nil {
				return err
			}
			// Reload the chart with the updated Chart.lock file.
			if chart, err = loader.Load(chartPath); err != nil {
				return fmt.Errorf("failed to reload chart after repo update: %v", err)
			}
		}
	}

	release, err := upgradeClient.RunWithContext(ctx, releaseName, chart, releaseValues)
	if err != nil {
		return fmt.Errorf("failed to run upgrade action: %v", err)
	}

	log.Debugf("release: %+v", release)

	return nil
}
