package k3s

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/infra/file"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

type Asset struct {
	Src        string `yaml:"src"`
	Dest       string `yaml:"dest"`
	Executable bool   `yaml:"executable"`
}

func Download(ks *spec.K3sSpec) error {
	assets := []Asset{
		{
			Src:        ks.BinaryURL(),
			Dest:       ks.BinaryPath(),
			Executable: true,
		},
		{
			Src:  ks.ImageURL(),
			Dest: ks.ImagePath(),
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

		if err := file.DownloadFile(URL.String(), asset.Dest); err != nil {
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
