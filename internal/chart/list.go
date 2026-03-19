package chart

import (
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"

	"github.com/Masterminds/semver/v3"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

func List(spec *spec.ChartSpec) ([]*semver.Version, error) {
	indexURL, err := url.JoinPath(spec.RepoURL, "index.yaml")
	if err != nil {
		return nil, err
	}

	resp, err := http.Get(indexURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index.yaml: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bad status %s for request: %s", resp.Status, spec.RepoURL)
	}

	indexFile := &repo.IndexFile{}
	body, _ := io.ReadAll(resp.Body)
	if err = yaml.Unmarshal(body, indexFile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal index.yaml: %w", err)
	}

	var versions []*semver.Version
	for chartName, chartVersions := range indexFile.Entries {
		if chartName == spec.ChartName {
			for _, chartVersion := range chartVersions {
				semverVersion, _ := semver.NewVersion(chartVersion.Metadata.Version)
				versions = append(versions, semverVersion)
			}
		}
	}

	sort.Sort(semver.Collection(versions))

	return versions, nil
}

func LastVersion(spec *spec.ChartSpec) *semver.Version {
	versions, err := List(spec)
	if err != nil {
		log.Fatalf("failed to list chart versions: %s", err)
		return nil
	}

	return versions[len(versions)-1]
}
