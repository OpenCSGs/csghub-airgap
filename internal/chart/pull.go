package chart

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	logger "log"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func Pull(spec *spec.ChartSpec) error {
	chartRef := spec.ChartRef()
	chartFileName := spec.ChartFileName()

	if _, err := os.Stat(spec.ChartFilePath()); err == nil {
		log.Infof("Chart %s already exists", chartFileName)
		return nil
	}

	settings := cli.New()
	repoFile := settings.RepositoryConfig
	rf, err := repo.LoadFile(repoFile)
	if err != nil || rf == nil {
		rf = repo.NewFile()
	}

	rf.Update(&repo.Entry{Name: spec.RepoName, URL: spec.RepoURL})
	if err = rf.WriteFile(repoFile, 0644); err != nil {
		return fmt.Errorf("failed to write repo config: %v", err)
	}

	entry := &repo.Entry{Name: spec.RepoName, URL: spec.RepoURL}
	chartRepo, err := repo.NewChartRepository(entry, getter.All(settings))
	if err != nil {
		return fmt.Errorf("failed to create chart repo for %s: %v", spec.RepoName, err)
	}

	if _, err = chartRepo.DownloadIndexFile(); err != nil {
		return fmt.Errorf("failed to download repo index for %s: %v", spec.RepoName, err)
	}

	chartDir := spec.ChartDir()
	if err = os.MkdirAll(chartDir, 0755); err != nil {
		return err
	}

	version := spec.ChartVersion
	if err = runPull(log.Std, settings, chartRef, version, chartDir); err != nil {
		return fmt.Errorf("failed to pull chart %s version %s: %v", chartRef, version, err)
	}

	log.Infof("Successfully pulled chart %s ", chartFileName)

	return nil
}

func runPull(logger *logger.Logger, settings *cli.EnvSettings, chartRef, chartVersion, destDir string) error {
	actionConfig, err := initActionConfig(settings, logger)
	if err != nil {
		return fmt.Errorf("failed to init action config: %w", err)
	}

	registryClient, err := newRegistryClient(settings, false)
	if err != nil {
		return fmt.Errorf("failed to created registry client: %w", err)
	}
	actionConfig.RegistryClient = registryClient

	pullClient := action.NewPullWithOpts(
		action.WithConfig(actionConfig))
	// client.RepoURL = ""
	pullClient.DestDir = destDir
	pullClient.Settings = settings
	pullClient.Version = chartVersion

	result, err := pullClient.Run(chartRef)
	if err != nil {
		return fmt.Errorf("failed to pull chart: %w", err)
	}

	log.Debugf("%+v", result)

	return nil
}
