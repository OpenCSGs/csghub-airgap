package chart

import (
	"context"
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

func Install(spec *spec.ChartSpec) error {
	ctx := context.Background()

	namespace := spec.RepoName

	// Initialize Helm settings
	settings := cli.New()
	settings.KubeConfig = "/etc/rancher/k3s/k3s.yaml"

	if namespace != "" {
		settings.SetNamespace(namespace)
	}

	chartRef := spec.ChartRef()
	chartVersion := spec.ChartVersion
	releaseName := spec.RepoName

	if releaseName == "" {
		return fmt.Errorf("release name must be specified")
	}

	log.Infof("Installing chart %s:%s as release %s", chartRef, chartVersion, releaseName)

	// Values (can be empty)
	values := map[string]interface{}{}
	if spec.ChartOptions != nil {
		values = spec.ChartOptions
	}

	// Run install
	if err := runInstall(ctx, log.Std, settings, releaseName, chartRef, chartVersion, values); err != nil {
		return err
	}

	logger.Printf("Install completed: %s", releaseName)
	return nil
}

func runInstall(ctx context.Context, logger *logger.Logger, settings *cli.EnvSettings, releaseName string, chartRef string, chartVersion string, releaseValues map[string]interface{}) error {
	actionConfig, err := initActionConfig(settings, logger)
	if err != nil {
		return fmt.Errorf("failed to init action config: %w", err)
	}

	installClient := action.NewInstall(actionConfig)

	installClient.DryRunOption = "none"
	installClient.ReleaseName = releaseName
	installClient.Namespace = settings.Namespace()
	installClient.Version = chartVersion
	installClient.CreateNamespace = true

	registryClient, err := newRegistryClientTLS(
		settings,
		logger,
		installClient.CertFile,
		installClient.KeyFile,
		installClient.CaFile,
		installClient.InsecureSkipTLSverify,
		installClient.PlainHTTP)
	if err != nil {
		return fmt.Errorf("failed to created registry client: %w", err)
	}
	installClient.SetRegistryClient(registryClient)

	chartPath, err := installClient.ChartPathOptions.LocateChart(chartRef, settings)
	if err != nil {
		return err
	}

	providers := getter.All(settings)

	chart, err := loader.Load(chartPath)
	if err != nil {
		return err
	}

	// Check chart dependencies to make sure all are present in /charts
	if chartDependencies := chart.Metadata.Dependencies; chartDependencies != nil {
		if err := action.CheckDependencies(chart, chartDependencies); err != nil {
			err = fmt.Errorf("failed to check chart dependencies: %w", err)
			if !installClient.DependencyUpdate {
				return err
			}

			manager := &downloader.Manager{
				Out:              logger.Writer(),
				ChartPath:        chartPath,
				Keyring:          installClient.ChartPathOptions.Keyring,
				SkipUpdate:       false,
				Getters:          providers,
				RepositoryConfig: settings.RepositoryConfig,
				RepositoryCache:  settings.RepositoryCache,
				Debug:            settings.Debug,
				RegistryClient:   installClient.GetRegistryClient(),
			}
			if err := manager.Update(); err != nil {
				return err
			}
			// Reload the chart with the updated Chart.lock file.
			if chart, err = loader.Load(chartPath); err != nil {
				return fmt.Errorf("failed to reload chart after repo update: %w", err)
			}
		}
	}

	release, err := installClient.RunWithContext(ctx, chart, releaseValues)
	if err != nil {
		return fmt.Errorf("failed to run install: %w", err)
	}

	log.Debugf("release created:\n%+v", *release)

	return nil
}
