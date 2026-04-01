package config

import (
	"csghub-airgap/internal/log"
	"fmt"
	"time"

	"github.com/urfave/cli/v2"
)

var DefaultBaseDir = fmt.Sprintf("csghub-airgap-%s", time.Now().Format("2006-01-02"))

func InitBaseDir(c *cli.Context) {
	baseDir := c.String("output")
	if baseDir == "" {
		baseDir = c.String("input")
	}
	if baseDir == "" {
		return
	}

	DefaultBaseDir = baseDir
	log.Debugf("Using baseDir: %s", baseDir)
}

func BaseDir() string {
	return DefaultBaseDir
}
