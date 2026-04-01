package command

import (
	"csghub-airgap/internal/app/bundle"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"

	"github.com/urfave/cli/v2"
)

func Create() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create csghub offline bundle",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Category: ">\u200BGlobal Settings",
				Usage:    "Output directory for the bundle",
				Value:    config.DefaultBaseDir,
			},
			&cli.GenericFlag{
				Name:     "arch",
				Aliases:  []string{"a"},
				Category: ">\u200BGlobal Settings",
				Usage:    "Target architecture for binaries and container images (e.g. amd64, arm64)",
				Value:    func() *Architecture { p := Architecture("amd64"); return &p }()},
			&cli.BoolFlag{
				Name:     "debug",
				Aliases:  []string{"d"},
				Category: ">\u200BGlobal Settings",
				Usage:    "Enable debug logging",
			},
			&cli.BoolFlag{
				Name:     "skip-k3s",
				Category: "> \u200CK3s Settings",
				Usage:    "Skip K3s installation",
			},
			&cli.StringSliceFlag{
				Name:     "k3s-env",
				Category: "> \u200CK3s Settings",
				Usage:    "Extra environment variables for K3s (KEY=VALUE)",
			},
			&cli.StringSliceFlag{
				Name:     "k3s-arg",
				Category: "> \u200CK3s Settings",
				Usage:    "Extra CLI arguments for K3s (--k3s-arg=--default-runtime=nvidia)",
			},
			&cli.StringFlag{
				Name:     "csghub-version",
				Aliases:  []string{"cv"},
				Category: "> \u200DCharts Settings",
				Usage:    "CSGHub chart version",
			},
			&cli.StringFlag{
				Name:     "longhorn-version",
				Aliases:  []string{"lv"},
				Category: "> \u200DCharts Settings",
				Usage:    "Longhorn chart version",
				Value:    "1.11.1",
			},
			&cli.BoolFlag{
				Name:     "skip-longhorn",
				Category: "> \u200DCharts Settings",
				Usage:    "Skip Longhorn installation",
			},
			&cli.StringFlag{
				Name:     "nvdp-version",
				Aliases:  []string{"nv"},
				Category: "> \u200DCharts Settings",
				Usage:    "NVIDIA device plugin version",
				Value:    "0.19.0",
			},
			&cli.BoolFlag{
				Name:     "skip-nvdp",
				Usage:    "Skip NVDP installation",
				Category: "> \u200DCharts Settings",
			},
			// ===== Helm values =====
			&cli.StringSliceFlag{
				Name:     "set",
				Aliases:  []string{"s"},
				Category: "> \u200DCharts Settings",
				Usage:    "Set Helm values (e.g. csghub.key=value, supports multiple)",
			},
		},
		Action: func(c *cli.Context) error {
			// init logging
			log.Init(c.Bool("debug"))

			// Load manifest from YAMLs
			config.InitBaseDir(c)
			config.InitArchitecture(c)

			// load spec
			bs, err := spec.LoadBundleSpec()
			if err != nil {
				return err
			}

			// apply CLI overrides
			if err := ApplyOverrides(bs, c); err != nil {
				return err
			}

			// build bundle
			return bundle.NewFromSpec(bs).Create()
		},
	}
}
