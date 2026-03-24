package main

import (
	"csghub-airgap/internal/bundle"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/flag"
	"csghub-airgap/internal/log"
	"csghub-airgap/internal/spec"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "csghub-airgap",
		Usage:   "Create or Deploy CSGHub with airgap bundle",
		Version: "1.17.0",
		Commands: []*cli.Command{
			createCommand(),
			deployCommand(),
			cleanupCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("%v", err)
	}
}

func createCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create csghub offline bundle",
		Flags: []cli.Flag{
			// ===== Global =====
			&cli.StringFlag{
				Name:     "output",
				Aliases:  []string{"o"},
				Usage:    "Output bundle path",
				Required: false,
				Value:    config.DefaultDestDir,
			},
			// ===== K3S =====
			&cli.StringFlag{
				Name:    "k3s-version",
				Aliases: []string{"kv"},
				Usage:   "K3s version",
				EnvVars: []string{"INSTALL_K3S_VERSION"},
				Value:   "v1.34.5+k3s1",
			},
			&cli.BoolFlag{
				Name:    "skip-k3s",
				Aliases: []string{"sk"},
				Usage:   "Skip k3s offline",
				Value:   false,
			},
			//&cli.StringSliceFlag{
			//	Name:    "k3s-env",
			//	Aliases: []string{"ke"},
			//	Usage:   "Set k3s install environments",
			//},
			//&cli.StringSliceFlag{
			//	Name:    "k3s-arg",
			//	Aliases: []string{"ka"},
			//	Usage:   "Set k3s install arguments",
			//},

			// ===== CSGHub =====
			&cli.StringFlag{
				Name:    "csghub-version",
				Aliases: []string{"cv"},
				Usage:   "Csghub chart version",
			},

			// ===== Longhorn =====
			&cli.StringFlag{
				Name:    "longhorn-version",
				Aliases: []string{"lv"},
				Usage:   "Longhorn chart version",
				Value:   "1.11.1",
			},
			&cli.BoolFlag{
				Name:    "skip-longhorn",
				Aliases: []string{"sl"},
				Usage:   "Skip longhorn offline",
				Value:   false,
			},

			// ===== Nvdp =====
			&cli.StringFlag{
				Name:    "nvdp-version",
				Aliases: []string{"nv"},
				Usage:   "Nvidia device plugin version",
				Value:   "0.19.0",
			},
			&cli.BoolFlag{
				Name:    "skip-nvdp",
				Aliases: []string{"sn"},
				Usage:   "Skip nvdp offline",
				Value:   false,
			},

			// ===== All Charts =====
			&cli.StringSliceFlag{
				Name:  "set",
				Usage: "Set chart values (e.g. csghub.global.image.registry=xxx)",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Turn on debug mode",
				Value:   false,
			},
		},
		Action: func(c *cli.Context) error {
			debug := c.Bool("debug")
			log.Init(debug)

			var opts []spec.BundleSpecOption

			opts = append(opts, spec.WithDestDir(c.String("output")))

			// ===== k3s =====
			if v := c.String("k3s-version"); v != "" {
				opts = append(opts, spec.WithK3sVersion(v))
			}

			opts = append(opts, spec.WithK3sSkip(c.Bool("skip-k3s")))

			//if k3sEnvs := c.StringSlice("k3s-env"); len(k3sEnvs) > 0 {
			//	opts = append(opts, spec.WithK3sEnvs(k3sEnvs))
			//}
			//
			//if k3sArgs := c.StringSlice("k3s-arg"); len(k3sArgs) > 0 {
			//	opts = append(opts, spec.WithK3sArgs(k3sArgs))
			//}

			// ===== chart version =====
			// ===== CSGHub =====
			if v := c.String("csghub-version"); v != "" {
				opts = append(opts, spec.WithCsghubVersion(v))
			}

			// ===== Longhorn =====
			if v := c.String("longhorn-version"); v != "" {
				opts = append(opts, spec.WithLonghornVersion(v))
			}

			opts = append(opts, spec.WithLonghornSkip(c.Bool("skip-longhorn")))

			// ===== NVDP =====
			if v := c.String("nvdp-version"); v != "" {
				opts = append(opts, spec.WithNvdpVersion(v))
			}

			opts = append(opts, spec.WithNvdpSkip(c.Bool("skip-nvdp")))

			// ===== chart options =====
			if sets := c.StringSlice("set"); len(sets) > 0 {
				all := flag.ParseSetFlags(sets)

				csghubOptions := flag.ExtractByPrefix(all, "csghub")
				opts = append(opts, spec.WithCsghubOptions(csghubOptions))

				longhornOptions := flag.ExtractByPrefix(all, "longhorn")
				opts = append(opts, spec.WithLonghornOptions(longhornOptions))

				nvdpOptions := flag.ExtractByPrefix(all, "nvdp")
				opts = append(opts, spec.WithNvdpOptions(nvdpOptions))
			}

			bs, err := spec.NewBundleSpec(opts...)
			if err != nil {
				return err
			}

			bundleSpec := bundle.NewFromSpec(bs)
			if err = bundleSpec.Create(); err != nil {
				return fmt.Errorf("create csghub bundle: %w", err)
			}

			return nil
		},
	}
}

func deployCommand() *cli.Command {
	return &cli.Command{
		Name:  "deploy",
		Usage: "Deploy csghub from offline bundle",
		Flags: []cli.Flag{
			// ===== Global =====
			&cli.StringFlag{
				Name:     "input",
				Aliases:  []string{"i"},
				Usage:    "Input bundle path",
				Required: false,
				Value:    config.DefaultDestDir,
			},
			// ===== K3S =====
			&cli.BoolFlag{
				Name:    "skip-k3s",
				Aliases: []string{"sk"},
				Usage:   "Skip k3s offline",
				Value:   false,
			},
			&cli.StringSliceFlag{
				Name:    "k3s-env",
				Aliases: []string{"ke"},
				Usage:   "Set k3s install environments",
			},
			&cli.StringSliceFlag{
				Name:    "k3s-arg",
				Aliases: []string{"ka"},
				Usage:   "Set k3s install arguments",
			},

			// ===== CSGHub =====
			&cli.StringFlag{
				Name:    "csghub-domain",
				Aliases: []string{"cd"},
				Usage:   "Specify csghub domain",
				Value:   "csghub.example.com",
			},

			// ===== Longhorn =====
			&cli.BoolFlag{
				Name:    "skip-longhorn",
				Aliases: []string{"sl"},
				Usage:   "Skip longhorn offline",
				Value:   false,
			},

			// ===== Nvdp =====
			&cli.BoolFlag{
				Name:    "skip-nvdp",
				Aliases: []string{"sn"},
				Usage:   "Skip nvdp offline",
				Value:   true,
			},

			// ===== All Charts =====
			&cli.StringSliceFlag{
				Name:  "set",
				Usage: "Set chart values (e.g. csghub.global.image.registry=xxx)",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "Turn on debug mode",
				Value:   false,
			},
		},
		Action: func(c *cli.Context) error {
			debug := c.Bool("debug")
			log.Init(debug)

			var opts []spec.BundleSpecOption

			inputDir := c.String("input")
			opts = append(opts, spec.WithDestDir(inputDir))

			// ===== k3s =====
			opts = append(opts, spec.WithK3sSkip(c.Bool("skip-k3s")))

			if k3sEnvs := c.StringSlice("k3s-env"); len(k3sEnvs) > 0 {
				opts = append(opts, spec.WithK3sEnvs(k3sEnvs))
			}

			if k3sArgs := c.StringSlice("k3s-arg"); len(k3sArgs) > 0 {
				opts = append(opts, spec.WithK3sArgs(k3sArgs))
			}

			// ===== Charts =====
			// ===== CSGHub =====
			csghubOptions := map[string]interface{}{
				"global": map[string]interface{}{
					"gateway": map[string]interface{}{
						"external": map[string]interface{}{
							"domain": c.String("csghub-domain"),
						},
					},
				},
			}
			opts = append(opts, spec.WithCsghubOptions(csghubOptions))

			// ===== Longhorn =====
			opts = append(opts, spec.WithLonghornSkip(c.Bool("skip-longhorn")))

			// ===== NVDP =====
			opts = append(opts, spec.WithNvdpSkip(c.Bool("skip-nvdp")))

			// ===== chart options =====
			if sets := c.StringSlice("set"); len(sets) > 0 {
				all := flag.ParseSetFlags(sets)

				csghubOptions = flag.ExtractByPrefix(all, "csghub")
				opts = append(opts, spec.WithCsghubOptions(csghubOptions))

				longhornOptions := flag.ExtractByPrefix(all, "longhorn")
				opts = append(opts, spec.WithLonghornOptions(longhornOptions))

				nvdpOptions := flag.ExtractByPrefix(all, "nvdp")
				opts = append(opts, spec.WithNvdpOptions(nvdpOptions))
			}

			bs, err := spec.NewBundleSpec(opts...)
			if err != nil {
				return err
			}

			bundleSpec, err := bundle.LoadFromManifest(inputDir, bs)
			if err != nil {
				return fmt.Errorf("failed to load bundle manifest: %w", err)
			}

			if err = bundleSpec.Deploy(); err != nil {
				return fmt.Errorf("%w", err)
			}

			return nil
		},
	}
}

func cleanupCommand() *cli.Command {
	return &cli.Command{
		Name:  "cleanup",
		Usage: "Cleanup k3s and all services",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation and run cleanup directly",
				Value:   false,
			},
		},
		Action: func(c *cli.Context) error {
			yes := c.Bool("yes")
			if !yes {
				fmt.Print("This will delete all CSGHub resources. Continue? (y/N): ")
				var input string
				_, err := fmt.Scanln(&input)
				if err != nil || (input != "y" && input != "Y") {
					fmt.Println("Cleanup aborted.")
					return nil
				}
			}

			return bundle.Cleanup()
		},
	}
}
