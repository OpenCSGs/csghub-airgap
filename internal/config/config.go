package config

import (
	"csghub-airgap/internal/log"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

const ProxyPrefix = "https://ghfast.top/"

var (
	DefaultBaseDir      = fmt.Sprintf("csghub-airgap-%s", time.Now().Format("2006-01-02"))
	DefaultArchitecture = runtime.GOARCH
	UseProxy            = false
)

// MaybeWrapProxy prepends the ghfast.top proxy prefix to a GitHub URL when
// proxying is enabled. Non-GitHub URLs are returned unchanged.
func MaybeWrapProxy(rawURL string) string {
	if !UseProxy {
		return rawURL
	}
	if strings.HasPrefix(rawURL, ProxyPrefix) {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "https://github.com/") ||
		strings.HasPrefix(rawURL, "https://raw.githubusercontent.com/") {
		return ProxyPrefix + rawURL
	}
	return rawURL
}

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

func InitProxy(c *cli.Context) {
	UseProxy = c.Bool("proxy")
	log.Debugf("Using proxy: %v", UseProxy)
}
