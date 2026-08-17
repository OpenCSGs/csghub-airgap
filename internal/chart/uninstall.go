package chart

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	logger "log"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
)

func Uninstall(spec *spec.ChartSpec) error {
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

	releaseName := spec.RepoName
	if releaseName == "" {
		return fmt.Errorf("release name must be specified")
	}

	log.Infof("Uninstalling release %s", releaseName)

	// Run uninstall
	if err := runUninstall(log.Std, settings, releaseName); err != nil {
		return err
	}

	log.Infof("Uninstall completed: %s", releaseName)
	return nil
}

func runUninstall(logger *logger.Logger, settings *cli.EnvSettings, releaseName string) error {
	actionConfig, err := initActionConfig(settings, logger)
	if err != nil {
		return fmt.Errorf("failed to init action config: %v", err)
	}

	uninstallClient := action.NewUninstall(actionConfig)
	uninstallClient.DeletionPropagation = "foreground" // "background" or "orphan"

	result, err := uninstallClient.Run(releaseName)
	if err != nil {
		return fmt.Errorf("failed to run uninstall action: %v", err)
	}
	if result != nil {
		logger.Printf("result: %+v\n", *result.Release)
	}

	log.Infof("release \"%s\" uninstalled\n", releaseName)

	return nil
}
