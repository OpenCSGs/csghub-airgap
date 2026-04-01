package main

import (
	"csghub-airgap/internal/app/command"
	"csghub-airgap/internal/log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "csghub-airgap",
		Usage:   "Create or Deploy CSGHub with airgap bundle",
		Version: "1.18.0",
		Commands: []*cli.Command{
			command.Create(),
			command.Deploy(),
			command.Cleanup(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatalf("%v", err)
	}
}
