package command

import (
	"context"
	"csghub-airgap/internal/image"
	"csghub-airgap/internal/log"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/gosuri/uitable"
	"github.com/urfave/cli/v2"
)

func ListImages() *cli.Command {
	return &cli.Command{
		Name:  "list-images",
		Usage: "List runtime images from csghub-server configs",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "source",
				Usage:    "Where to fetch configs: github, local (local directory for air-gapped use)",
				Value:    "github",
				Category: "Query Options",
			},
			&cli.StringFlag{
				Name:     "version",
				Aliases:  []string{"v"},
				Usage:    "Git branch or tag name for csghub-server (e.g. main, v1.2.0)",
				Value:    "main",
				Category: "Query Options",
			},
			&cli.StringFlag{
				Name:     "local-path",
				Usage:    "Local directory containing csghub-server configs (used with --source local)",
				Value:    ".",
				Category: "Query Options",
			},
			&cli.StringSliceFlag{
				Name:     "category",
				Aliases:  []string{"c"},
				Usage:    "Filter by category: evaluation, finetune, inference, job, notebook, space (can specify multiple)",
				Category: "Query Options",
			},
			&cli.StringFlag{
				Name:     "compute",
				Usage:    "Filter by compute type: cpu, gpu, dcu (can specify multiple, comma-separated)",
				Category: "Query Options",
			},
			&cli.StringFlag{
				Name:     "format",
				Aliases:  []string{"f"},
				Usage:    "Output format: table, json",
				Value:    "table",
				Category: "Output Options",
			},
		},
		Action: func(c *cli.Context) error {
			log.Init(false)

			source := image.Source(c.String("source"))
			version := c.String("version")
			localPath := c.String("local-path")
			categories := c.StringSlice("category")

			var computeFilter []string
			if compute := c.String("compute"); compute != "" {
				for _, ct := range splitCSV(compute) {
					computeFilter = append(computeFilter, ct)
				}
			}

			// For local source, use local-path as the path argument
			ref := version
			if source == image.SourceLocal {
				ref = localPath
			}

			ctx := context.Background()
			images, err := image.ListRuntimeImages(ctx, source, ref, categories, computeFilter)
			if err != nil {
				return fmt.Errorf("failed to list runtime images: %w", err)
			}

			if len(images) == 0 {
				log.Warn("no runtime images found for the specified filters")
				return nil
			}

			format := c.String("format")
			switch format {
			case "json":
				return printJSON(images)
			case "table":
				printTable(images)
				return nil
			default:
				return fmt.Errorf("unknown format: %s (supported: table, json)", format)
			}
		},
	}
}

func printTable(images []image.RuntimeImageInfo) {
	table := uitable.New()
	table.MaxColWidth = 80

	table.AddRow("CATEGORY", "ENGINE", "COMPUTE", "IMAGE", "ENGINE VER", "DRIVER VER")
	table.AddRow("--------", "------", "-------", "-----", "----------", "----------")

	for _, img := range images {
		table.AddRow(img.Category, img.EngineName, img.ComputeType, img.Image, img.EngineVer, img.DriverVer)
	}

	fmt.Fprintln(os.Stdout, table.String())
	fmt.Fprintf(os.Stderr, "\nFound %d images\n", len(images))
}

func printJSON(images []image.RuntimeImageInfo) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(images)
}

func splitCSV(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
