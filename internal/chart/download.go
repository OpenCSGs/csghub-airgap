package chart

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	"os"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/repo"
)

func Download(spec *spec.ChartSpec) error {
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
