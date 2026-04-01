package net

import "csghub-airgap/internal/infra/shell"

func SetDefaultRoute() error {
	cmdStr := `
ip link show dummy0 >/dev/null 2>&1 || (ip link add dummy0 type dummy &&
ip link set dummy0 up &&
ip addr add 203.0.113.254/31 dev dummy0 &&
ip route add default via 203.0.113.255 dev dummy0 metric 1000)
`

	_ = shell.Execute("bash", "-c", cmdStr)
	return nil
}
