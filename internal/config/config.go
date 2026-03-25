package config

import (
	"fmt"
	"time"
)

var (
	DefaultDestDir               = fmt.Sprintf("csghub-airgap-%s", time.Now().Format("2006-01-02"))
	DefaultRegistry              = "opencsg-registry.cn-beijing.cr.aliyuncs.com"
	DefaultNamespace             = "opencsghq"
	DefaultRegistryWithNamespace = fmt.Sprintf("%s/%s", DefaultRegistry, DefaultNamespace)
)
