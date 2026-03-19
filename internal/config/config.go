package config

import (
	"fmt"
	"time"
)

var DefaultDestDir = fmt.Sprintf("csghub-airgap-%s", time.Now().Format("2006-01-02"))
