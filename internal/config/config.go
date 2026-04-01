package config

import (
	"csghub-airgap/internal/log"
	"fmt"
	"runtime"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	DefaultBaseDir      = fmt.Sprintf("csghub-airgap-%s", time.Now().Format("2006-01-02"))
	DefaultArchitecture = runtime.GOARCH
)

func InitBaseDir(c *cli.Context) {
	var baseDir string

	if c.IsSet("output") {
		baseDir = c.String("output")
		log.Debugf("Using baseDir (from output): %s", baseDir)
	} else if c.IsSet("input") {
		baseDir = c.String("input")
		log.Debugf("Using baseDir (from input): %s", baseDir)
	} else {
		log.Debugf("Using baseDir (default): %s", DefaultBaseDir)
		return
	}

	if baseDir == "" {
		return
	}

	DefaultBaseDir = baseDir
}

func InitArchitecture(c *cli.Context) {
	if !c.IsSet("arch") {
		return
	}

	DefaultArchitecture = c.String("arch")
	log.Debugf("Using architecture: %s", DefaultArchitecture)
}
