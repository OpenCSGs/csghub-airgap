package command

import (
	"context"
	"csghub-airgap/internal/app/bundle"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"runtime"

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
				Category: ">​Global Settings",
				Usage:    "Output directory for the bundle",
				Value:    config.DefaultBaseDir,
			},
			&cli.GenericFlag{
				Name:     "arch",
				Aliases:  []string{"a"},
				Category: ">​Global Settings",
				Usage:    "Target architecture for binaries and container images (e.g. amd64, arm64)",
				Value:    func() *Architecture { p := Architecture(runtime.GOARCH); return &p }()},
			&cli.BoolFlag{
				Name:     "debug",
				Aliases:  []string{"d"},
				Category: ">​Global Settings",
				Usage:    "Enable debug logging",
			},
			&cli.BoolFlag{
				Name:     "proxy",
				Category: ">​Global Settings",
				Usage:    "Use ghfast.top proxy prefix for GitHub downloads",
			},
			&cli.BoolFlag{
				Name:     "force-pull",
				Aliases:  []string{"fp"},
				Category: ">​Global Settings",
				Usage:    "Skip local containerd image cache and always re-pull from registry",
				Value:    false,
			},
			&cli.BoolFlag{
				Name:     "skip-k3s",
				Category: "> ‌K3s Settings",
				Usage:    "Skip K3s installation",
			},
			&cli.StringSliceFlag{
				Name:     "k3s-env",
				Category: "> ‌K3s Settings",
				Usage:    "Extra environment variables for K3s (KEY=VALUE)",
			},
			&cli.StringSliceFlag{
				Name:     "k3s-arg",
				Category: "> ‌K3s Settings",
				Usage:    "Extra CLI arguments for K3s (--k3s-arg=--default-runtime=nvidia)",
			},
			&cli.StringFlag{
				Name:     "csghub-version",
				Aliases:  []string{"cv"},
				Category: "> ‍Charts Settings",
				Usage:    "CSGHub chart version",
			},
			&cli.BoolFlag{
				Name:     "csghub-runtime",
				Category: "> ‍Charts Settings",
				Usage:    "Interactively select CSGHub runtime images to include in the bundle",
				Value:    false,
			},
			&cli.StringFlag{
				Name:     "longhorn-version",
				Aliases:  []string{"lv"},
				Category: "> ‍Charts Settings",
				Usage:    "Longhorn chart version",
				Value:    "1.11.1",
			},
			&cli.BoolFlag{
				Name:     "skip-longhorn",
				Category: "> ‍Charts Settings",
				Usage:    "Skip Longhorn installation",
			},
			&cli.StringFlag{
				Name:     "nvdp-version",
				Aliases:  []string{"nv"},
				Category: "> ‍Charts Settings",
				Usage:    "NVIDIA device plugin version",
				Value:    "0.19.0",
			},
			&cli.BoolFlag{
				Name:     "skip-nvdp",
				Usage:    "Skip NVDP installation",
				Category: "> ‍Charts Settings",
			},
			// ===== Helm values =====
			&cli.StringSliceFlag{
				Name:     "set",
				Aliases:  []string{"s"},
				Category: "> ‍Charts Settings",
				Usage:    "Set Helm values (e.g. csghub.key=value, supports multiple)",
			},
		},
		Action: func(c *cli.Context) error {
			// init logging
			log.Init(c.Bool("debug"))

			// Propagate --force-pull to the image package so localImageExists()
			// bypasses the host containerd cache for this run.
			image.ForcePullImages = c.Bool("force-pull")

			// init config
			config.InitBaseDir(c)
			config.InitArchitecture(c)
			config.InitProxy(c)

			// load spec
			bs, err := spec.LoadBundleSpec()
			if err != nil {
				return err
			}

			// apply CLI overrides
			if err := ApplyOverrides(bs, c); err != nil {
				return err
			}

			// Interactive runtime image selection
			if bs.Runtime {
				if !c.Bool("debug") {
					log.Infof("Fetching available runtime images...")
				}
				ctx := context.Background()
				selected, err := image.SelectRuntimeImages(ctx, image.SourceGitHub, "main")
				if err != nil {
					return err
				}
				if selected == nil {
					log.Warn("Runtime image selection cancelled")
				} else {
					log.Infof("Selected %d runtime images", len(selected))
				}
				bs.SelectedRuntimeImages = selected
			}

			// build bundle
			return bundle.NewFromSpec(bs).Create()
		},
	}
}
