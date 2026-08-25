package bundle

// Pre-flight check for the multipathd service. Longhorn (the default
// storage backend for csghub) plays poorly with multipath: the kernel
// multipath daemon claims block devices and creates dm-* mappers that
// confuse Longhorn's replica discovery, leading to stuck or failed PVC
// mounts. The fix is to stop and disable multipathd on the host before
// the chart install loop runs.

import (
	"bufio"
	"fmt"
	osi "csghub-airgap/internal/infra/os"
	"os"
	"strings"

	"csghub-airgap/internal/log"
)

const multipathServiceName = "multipathd"

// checkMultipath inspects the host for an active multipathd service. When
// found and quiet is false, the operator is prompted to either stop+disable
// it (y) or continue with multipath running (n). When quiet is true, the
// service is stopped and disabled without asking.
//
// Returns nil when no action is needed (service inactive or absent) or when
// the chosen action completed successfully. Errors from systemctl are
// surfaced but never block the install — the operator can still proceed.
func checkMultipath(quiet bool) error {
	active, err := osi.IsServiceActive(multipathServiceName)
	if err != nil {
		log.Debugf("multipath check skipped: %v", err)
		return nil
	}
	if !active {
		log.Debugf("multipathd is not active; nothing to do")
		return nil
	}

	log.Warnf("multipathd is running on this host")
	log.Warnf("Longhorn (csghub's storage backend) does not coexist cleanly with multipath:")
	log.Warnf("  - multipath may claim block devices before Longhorn can use them")
	log.Warnf("  - PVCs may fail to mount or get stuck in Pending")
	log.Warnf("  - Longhorn replicas may report stale device paths")

	if quiet {
		log.Infof("--quiet set: stopping and disabling multipathd")
		return disableMultipath()
	}

	fmt.Print("Stop and disable multipathd now? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		log.Warnf("Could not read user input; continuing with multipath running")
		return nil
	}
	answer := strings.TrimSpace(strings.ToLower(line))
	if answer != "y" && answer != "yes" {
		log.Warnf("Continuing with multipathd running; PVC mounts may fail")
		return nil
	}
	return disableMultipath()
}

// disableMultipath stops the running service and prevents it from
// starting at boot. Errors from either step are logged but not returned
// so a stuck/half-broken service cannot block the install.
func disableMultipath() error {
	if err := osi.StopService(multipathServiceName); err != nil {
		log.Warnf("Failed to stop multipathd: %v", err)
		log.Warnf("Continuing; you may need to stop it manually before the chart install")
		return nil
	}
	if err := osi.DisableService(multipathServiceName); err != nil {
		log.Warnf("Failed to disable multipathd: %v", err)
		log.Warnf("multipathd was stopped but will restart on reboot")
		return nil
	}
	log.Infof("multipathd stopped and disabled")
	return nil
}
