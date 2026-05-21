package main

import (
	"csghub-airgap/internal/app/command"
	"csghub-airgap/internal/log"
	"os"

	"strings"

	"github.com/urfave/cli/v2"
)

var (
	Version = "dev"
)

func main() {
	version := Version

	if len(version) > 7 && !strings.Contains(version, ".") {
		version = version[:7]
	}

	app := &cli.App{
		Name:    "csghub-airgap",
		Usage:   "Create or Deploy CSGHub with airgap bundle",
		Version: version,
		Commands: []*cli.Command{
			command.Create(),
			command.Deploy(),
			command.Cleanup(),
			command.ListImages(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("%v", err)
	}
}
