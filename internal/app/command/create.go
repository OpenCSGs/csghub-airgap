package command

import (
	"context"
	"csghub-airgap/internal/app/bundle"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/infra/pkg"
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
				Category: ">​Global",
				Usage:    "Output directory for the bundle",
				Value:    config.DefaultBaseDir,
			},
			&cli.GenericFlag{
				Name:     "arch",
				Category: ">​Global",
				Usage:    "Target architecture for binaries and container images (e.g. amd64, arm64)",
				Value:    func() *Architecture { p := Architecture(runtime.GOARCH); return &p }()},
			&cli.BoolFlag{
				Name:     "debug",
				Aliases:  []string{"d"},
				Category: ">​Global",
				Usage:    "Enable debug logging",
			},
			&cli.BoolFlag{
				Name:     "proxy",
				Category: ">​Global",
				Usage:    "Use ghfast.top proxy prefix for GitHub downloads",
			},
			&cli.BoolFlag{
				Name:     "force-pull",
				Category: ">​Global",
				Usage:    "Skip local containerd image cache and always re-pull from registry",
				Value:    false,
			},
			&cli.BoolFlag{
				Name:     "no-pkg-cache",
				Category: ">​Global",
				Usage:    "Skip cached NVIDIA package directory and always re-download apt/dnf/zypper packages",
				Value:    false,
			},
			&cli.StringSliceFlag{
				Name:     "k3s-env",
				Category: "> ‌K3s",
				Usage:    "Extra environment variables for K3s (KEY=VALUE)",
			},
			&cli.StringSliceFlag{
				Name:     "k3s-arg",
				Category: "> ‌K3s",
				Usage:    "Extra CLI arguments for K3s (--k3s-arg=--default-runtime=nvidia)",
			},
			&cli.StringSliceFlag{
				Name:     "skip",
				Category: ">​Global",
				Usage:    "Skip components: comma-separated chart names (csghub, longhorn, nvdp) and/or kinds (k3s, image, runtime, package)",
			},
			&cli.BoolFlag{
				Name:     "quiet",
				Aliases:  []string{"q"},
				Category: "> Preflight",
				Usage:    "Skip interactive prompts. When set, runtime image selection is skipped (no images selected)",
			},
			// ===== Helm values =====
			&cli.StringSliceFlag{
				Name:     "set",
				Aliases:  []string{"s"},
				Category: "> Charts",
				Usage:    "Set Helm values (e.g. csghub.key=value, supports multiple)",
			},
		},
		Action: func(c *cli.Context) error {
			// init logging
			log.Init(c.Bool("debug"))

			// Propagate --force-pull to the image package so localImageExists()
			// bypasses the host containerd cache for this run.
			image.ForcePullImages = c.Bool("force-pull")

			// Propagate --no-pkg-cache to the pkg package so the NVIDIA package
			// download skips the cache lock check and always re-pulls.
			pkg.NoCache = c.Bool("no-pkg-cache")

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

			bs.Quiet = c.Bool("quiet")

			// Runtime image selection is on by default and skipped when
			// --skip runtime (or --skip image, which also sets
			// SkipRuntimeImages) is in effect, or when --quiet is set —
			// both mean "don't block on an interactive prompt".
			if !bs.SkipRuntimeImages && !bs.Quiet {
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
			} else if bs.SkipRuntimeImages {
				log.Infof("Skipping runtime image selection (--skip runtime/image)")
			} else {
				log.Infof("Skipping runtime image selection (--quiet)")
			}

			// build bundle
			return bundle.NewFromSpec(bs).Create()
		},
	}
}
