package k3s

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/infra/file"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Asset struct {
	Src        string `yaml:"src"`
	Dest       string `yaml:"dest"`
	Executable bool   `yaml:"executable"`
	Checksum   string `yaml:"checksum"`
}

func Download(ks *spec.K3sSpec) error {
	// The official release publishes sha256sum-<arch>.txt. Fetch the expected
	// hash for the airgap images so a corrupt/incomplete download is caught and
	// re-fetched instead of being shipped in the bundle. A missing checksum only
	// warns, so an unreachable checksum endpoint does not block create.
	imageSHA := ""
	if sha, err := fetchAirgapImageSHA256(ks); err != nil {
		log.Warnf("Skipping airgap image checksum verification: %v", err)
	} else {
		imageSHA = sha
	}

	assets := []Asset{
		{
			Src:        ks.BinaryURL(),
			Dest:       ks.BinaryPath(),
			Executable: true,
		},
		{
			Src:      ks.ImageURL(),
			Dest:     ks.ImagePath(),
			Checksum: imageSHA,
		},
		{
			Src:        config.MaybeWrapProxy(ks.ScriptURL),
			Dest:       ks.ScriptPath(),
			Executable: true,
		},
		{
			Src:        "https://charts.opencsg.com/repository/scripts/coredns-custom-patch.sh",
			Dest:       filepath.Join(filepath.Dir(ks.ScriptPath()), "k3s-patch.sh"),
			Executable: true,
		},
		{
			Src:        ks.HelmBinaryURL(),
			Dest:       ks.HelmBinaryPath(),
			Executable: true,
		},
	}

	log.DebugFunc(func() string {
		data, _ := json.Marshal(assets)
		return fmt.Sprintf("Downloading assets: %s", data)
	})

	for _, asset := range assets {
		URL, err := url.Parse(asset.Src)
		if err != nil {
			return fmt.Errorf("cannot parse url %s: %v", asset.Src, err)
		}

		if asset.Checksum != "" {
			if err := file.DownloadFileWithChecksum(URL.String(), asset.Dest, asset.Checksum); err != nil {
				return err
			}
		} else if err := file.DownloadFile(URL.String(), asset.Dest); err != nil {
			return err
		}

		if asset.Executable {
			if err := os.Chmod(asset.Dest, 0755); err != nil {
				return err
			}
		}
	}

	return nil
}

// fetchAirgapImageSHA256 downloads sha256sum-<arch>.txt from the k3s release and
// returns the hash for the airgap image archive. The file lists every release
// asset as "<hash>  <filename>" lines.
func fetchAirgapImageSHA256(ks *spec.K3sSpec) (string, error) {
	sumURL, err := url.JoinPath(ks.BaseURL(), fmt.Sprintf("sha256sum-%s.txt", config.DefaultArchitecture))
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(sumURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch checksum file %s: %v", sumURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch checksum file %s: %s", sumURL, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	target := ks.ImageName()
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == target {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("sha256 for %s not found in %s", target, sumURL)
}
