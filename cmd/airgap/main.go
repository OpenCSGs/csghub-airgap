package main

import (
	"csghub-airgap/internal/bundle"
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/flag"
	"csghub-airgap/internal/spec"
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "csghub-airgap",
		Usage:   "CSGHub offline bundle builder and installer",
		Version: "1.17.0",
		Commands: []*cli.Command{
			createCommand(),
			deployCommand(),
			cleanupCommand(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
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
				Usage:    "output bundle path",
				Required: false,
				Value:    config.DefaultDestDir,
			},
			// ===== K3S =====
			&cli.StringFlag{
				Name:    "k3s-version",
				Aliases: []string{"kv"},
				Usage:   "k3s version",
				Value:   "v1.34.5+k3s1",
			},
			// ===== CSGHub =====
			&cli.StringFlag{
				Name:    "csghub-version",
				Aliases: []string{"cv"},
				Usage:   "csghub chart version",
			},
			&cli.StringFlag{
				Name:    "longhorn-version",
				Aliases: []string{"lv"},
				Usage:   "longhorn chart version",
			},
			&cli.StringFlag{
				Name:    "nvdp-version",
				Aliases: []string{"nv"},
				Usage:   "nvidia device plugin version",
			},
			&cli.StringSliceFlag{
				Name:  "set",
				Usage: "set chart values (e.g. csghub.global.image.registry=xxx)",
			},
		},
		Action: func(c *cli.Context) error {
			var opts []spec.BundleSpecOption

			opts = append(opts, spec.WithDestDir(c.String("output")))

			// ===== k3s =====
			if v := c.String("k3s-version"); v != "" {
				opts = append(opts, spec.WithK3sVersion(v))
			}

			// ===== chart version =====
			if v := c.String("csghub-version"); v != "" {
				opts = append(opts, spec.WithCsghubVersion(v))
			}
			if v := c.String("longhorn-version"); v != "" {
				opts = append(opts, spec.WithLonghornVersion(v))
			}
			if v := c.String("nvdp-version"); v != "" {
				opts = append(opts, spec.WithNvdpVersion(v))
			}

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
			if err := bundleSpec.Create(); err != nil {
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
			&cli.StringFlag{
				Name:     "bundle",
				Usage:    "offline bundle path",
				Required: true,
			},

			// k3s
			&cli.BoolFlag{
				Name:  "install-k3s",
				Usage: "install k3s automatically",
			},
			&cli.StringFlag{
				Name:  "k3s-version",
				Usage: "k3s version",
				Value: "v1.30.0+k3s1",
			},

			// longhorn
			&cli.BoolFlag{
				Name:  "install-longhorn",
				Usage: "install longhorn storage",
			},
			&cli.StringFlag{
				Name:  "longhorn-version",
				Usage: "longhorn version",
			},

			// csghub
			&cli.StringFlag{
				Name:  "csghub-version",
				Usage: "csghub version",
			},
			&cli.StringFlag{
				Name:  "csghub-domain",
				Usage: "csghub ingress domain",
			},
			&cli.IntFlag{
				Name:  "csghub-replicas",
				Usage: "csghub replicas",
				Value: 1,
			},
		},
		Action: func(c *cli.Context) error {

			bundle := c.String("bundle")

			fmt.Println("Deploying from bundle:", bundle)

			if c.Bool("install-k3s") {
				fmt.Println("Installing k3s:", c.String("k3s-version"))
			}

			if c.Bool("install-longhorn") {
				fmt.Println("Installing longhorn:", c.String("longhorn-version"))
			}

			fmt.Println("Deploying csghub")
			fmt.Println("Version:", c.String("csghub-version"))
			fmt.Println("Domain:", c.String("csghub-domain"))
			fmt.Println("Replicas:", c.Int("csghub-replicas"))

			// TODO: 执行部署逻辑
			return nil
		},
	}
}

func cleanupCommand() *cli.Command {
	return &cli.Command{
		Name:  "cleanup",
		Usage: "Cleanup csghub deployment",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "remove-k3s",
				Usage: "remove k3s cluster",
			},
			&cli.BoolFlag{
				Name:  "remove-longhorn",
				Usage: "remove longhorn",
			},
		},
		Action: func(c *cli.Context) error {

			fmt.Println("Cleaning up deployment")

			if c.Bool("remove-longhorn") {
				fmt.Println("Removing longhorn")
			}

			if c.Bool("remove-k3s") {
				fmt.Println("Removing k3s")
			}

			// TODO: cleanup logic

			return nil
		},
	}
}
