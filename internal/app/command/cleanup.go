package command

import (
	"csghub-airgap/internal/infra/shell"
	"csghub-airgap/internal/infra/user"
	"fmt"

	"github.com/urfave/cli/v2"
)

func Cleanup() *cli.Command {
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
			if err := user.EnsureRoot(); err != nil {
				return err
			}

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

			return shell.Execute("/usr/local/bin/k3s-uninstall.sh")
		},
	}
}
