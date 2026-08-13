package chart

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	"io"
	logger "log"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

func List(spec *spec.ChartSpec) bool {
	namespace := spec.RepoName

	// Initialize Helm settings
	settings := cli.New()
	settings.KubeConfig = "/etc/rancher/k3s/k3s.yaml"

	if namespace != "" {
		settings.SetNamespace(namespace)
	}

	log.Infof("Listing releases in namespace: %s", settings.Namespace())

	// Run list action
	releaseInstalled, err := runList(log.Std, settings)
	if err != nil {
		log.Fatalf("Error listing releases: %v", err)
		return false
	}

	if len(releaseInstalled) > 0 {
		return true
	}

	return false
}

func runList(logger *logger.Logger, settings *cli.EnvSettings) ([]*release.Release, error) {

	actionConfig, err := initActionConfigList(settings, logger, false)
	if err != nil {
		return nil, fmt.Errorf("failed to init action config: %v", err)
	}

	listClient := action.NewList(actionConfig)
	// Only list deployed
	//listClient.Deployed = true
	listClient.All = true
	listClient.SetStateMask()

	results, err := listClient.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run list action: %v", err)
	}

	log.Debugf("Found %d releases", len(results))

	return results, nil
}

func ListChart(spec *spec.ChartSpec) ([]*semver.Version, error) {
	indexURL, err := url.JoinPath(spec.RepoURL, "index.yaml")
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index.yaml: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status %s for request: %s", resp.Status, spec.RepoURL)
	}

	indexFile := &repo.IndexFile{}
	body, _ := io.ReadAll(resp.Body)
	if err = yaml.Unmarshal(body, indexFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index.yaml: %v", err)
	}

	var versions []*semver.Version
	for chartName, chartVersions := range indexFile.Entries {
		if chartName == spec.ChartName {
			for _, chartVersion := range chartVersions {
				semverVersion, err := semver.NewVersion(chartVersion.Metadata.Version)
				if err != nil {
					continue
				}
				versions = append(versions, semverVersion)
			}
		}
	}

	sort.Sort(semver.Collection(versions))

	return versions, nil
}

func LatestVersion(spec *spec.ChartSpec) string {
	versions, err := ListChart(spec)
	if err != nil {
		log.Fatalf("failed to list chart versions: %s", err)
		return ""
	}

	if len(versions) == 0 {
		log.Fatalf("no versions found for chart %s", spec.ChartName)
		return ""
	}

	latestVersion := versions[len(versions)-1]

	return latestVersion.String()
}
