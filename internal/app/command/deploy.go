package command

import (
	"csghub-airgap/internal/app/bundle"
	"csghub-airgap/internal/app/metadata"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/infra/user"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/utils/extract"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urfave/cli/v2"
)

func Deploy() *cli.Command {
	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy csghub from offline bundle",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "input",
				Aliases:  []string{"i"},
				Category: ">\u200BGlobal",
				Usage:    "Input bundle directory",
				Value:    config.DefaultBaseDir,
			},
			&cli.BoolFlag{
				Name:     "debug",
				Aliases:  []string{"d"},
				Category: ">\u200BGlobal",
				Usage:    "Enable debug logging",
			},
			&cli.BoolFlag{
				Name:     "proxy",
				Category: ">\u200BGlobal",
				Usage:    "Use ghfast.top proxy prefix for GitHub downloads",
			},
			&cli.StringSliceFlag{
				Name:     "k3s-env",
				Category: "> \u200CK3s",
				Usage:    "Extra environment variables for K3s (KEY=VALUE)",
			},
			&cli.StringSliceFlag{
				Name:     "k3s-arg",
				Category: "> \u200CK3s",
				Usage:    "Extra CLI arguments for K3s (--k3s-arg=--default-runtime=nvidia)",
			},
			&cli.StringFlag{
				Name:     "data",
				Aliases:  []string{"D"},
				Category: "> \u200CK3s",
				Usage:    "Specify data directory for K3s and Longhorn",
			},
			&cli.StringFlag{
				Name:     "domain",
				Category: "> Charts",
				Usage:    "Specify csghub domain",
				Value:    "csghub.example.com",
			},
			&cli.BoolFlag{
				Name:     "quiet",
				Aliases:  []string{"q"},
				Category: "> Preflight",
				Usage:    "Skip interactive prompts. When set, multipathd is auto-disabled (instead of prompting)",
			},
			&cli.StringSliceFlag{
				Name:     "skip",
				Category: ">​Global",
				Usage:    "Skip components: comma-separated chart names (csghub, longhorn, nvdp) and/or kinds (k3s, charts, image, runtime, package, tar)",
			},
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

			// init config
			config.InitBaseDir(c)
			config.InitProxy(c)

			// If --input points at the bundle archive produced by `create`,
			// unpack it before loading metadata. Skipped with --skip tar (the
			// input is then expected to be an already-unpacked directory).
			if !parseSkips(c.StringSlice("skip")).Tar {
				if dir, err := maybeUnpackBundle(config.DefaultBaseDir); err != nil {
					return err
				} else if dir != "" {
					log.Infof("Unpacked bundle archive to %s", dir)
					config.DefaultBaseDir = dir
				}
			}

			// Run as user root
			if err := user.EnsureRoot(); err != nil {
				return err
			}

			// load spec
			bs, err := metadata.Load()
			if err != nil {
				return err
			}

			if bs.Arch != runtime.GOARCH {
				log.Fatalf("Architecture mismatch: bundle requires %s, but current system is %s", bs.Arch, runtime.GOARCH)
			}

			bs.Quiet = c.Bool("quiet")

			// apply CLI overrides
			if err := ApplyOverrides(bs, c); err != nil {
				return err
			}

			// build bundle
			return bundle.NewFromSpec(bs).Deploy()
		},
	}
}

// maybeUnpackBundle unpacks a bundle tar when input points at one, returning
// the path of the unpacked directory. A plain directory (or a path that is not
// a .tar archive) is returned as "" so the caller keeps using input as-is.
func maybeUnpackBundle(input string) (string, error) {
	fi, err := os.Stat(input)
	if err != nil {
		return "", nil
	}
	if fi.IsDir() || !strings.HasSuffix(input, ".tar") {
		return "", nil
	}

	dest := strings.TrimSuffix(input, ".tar")
	log.Infof("Unpacking bundle archive %s ...", input)
	if err := extract.Untar(input, filepath.Dir(input)); err != nil {
		return "", fmt.Errorf("failed to unpack bundle %s: %w", input, err)
	}
	return dest, nil
}
