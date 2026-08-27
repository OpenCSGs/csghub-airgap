package bundle

// Pre-flight check for the multipathd service. Longhorn (the default
// storage backend for csghub) plays poorly with multipath: the kernel
// multipath daemon claims block devices and creates dm-* mappers that
// confuse Longhorn's replica discovery, leading to stuck or failed PVC
// mounts. The fix is to stop and disable multipathd on the host before
// the chart install loop runs.

import (
	"bufio"
	"csghub-airgap/internal/config"
	osi "csghub-airgap/internal/infra/os"
	"fmt"
	"os"
	"strings"

	"csghub-airgap/internal/log"
)

const (
	// multipathd is the systemd service name; `multipathd.socket` is the
	// socket-activation unit that re-triggers multipathd when a client
	// connects to /org/kernel/linux/storage/multipathd. Both must be
	// stopped+disabled together — leaving the socket active lets systemd
	// restart multipathd behind the operator's back. All mainstream distros
	// (Debian/Ubuntu, RHEL/CentOS/Fedora, openSUSE/SLES) ship the same two
	// units from upstream multipath-tools.
	multipathServiceName = "multipathd"
	multipathSocketName  = "multipathd.socket"
)

// multipathUnits are the systemd units to stop and disable in order. The
// service is listed first so the daemon is gone before the socket is
// retired; the socket may be absent on hosts that did not install it, and
// is therefore tolerated as a no-op.
var multipathUnits = []string{multipathServiceName, multipathSocketName}

// checkMultipath inspects the host for an active multipathd service or
// socket. When found and quiet is false, the operator is prompted to either
// stop+disable multipath (y) or continue with multipath running (n). When
// quiet is true, multipath is stopped and disabled without asking.
//
// Returns nil when no action is needed (nothing active) or when the chosen
// action completed. Errors from systemctl are surfaced but never block the
// install — the operator can still proceed.
func checkMultipath(quiet bool) error {
	anyActive := false
	for _, unit := range multipathUnits {
		active, err := osi.IsServiceActive(unit)
		if err != nil {
			log.Debugf("multipath unit %s check skipped: %v", unit, err)
			continue
		}
		if active {
			anyActive = true
			break
		}
	}
	if !anyActive {
		log.Debugf("multipathd is not active; nothing to do")
		return nil
	}

	log.Warnf("multipathd is running on this host")
	log.Warnf("Longhorn (csghub's storage backend) does not coexist cleanly with multipath:")
	log.Warnf("  - multipath may claim block devices before Longhorn can use them")
	log.Warnf("  - PVCs may fail to mount or get stuck in Pending")
	log.Warnf("  - Longhorn replicas may report stale device paths")

	if config.DryRun {
		log.Infof("[DRY-RUN] would stop and disable multipathd: %s", strings.Join(multipathUnits, ", "))
		return nil
	}

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

// disableMultipath stops and disables every multipath systemd unit. A
// missing unit (e.g. the socket not installed on a minimal host) is
// tolerated as a no-op; real errors are logged but never block the install.
func disableMultipath() error {
	for _, unit := range multipathUnits {
		if err := osi.StopService(unit); err != nil {
			if !isMissingUnit(err) {
				log.Warnf("Failed to stop %s: %v", unit, err)
				log.Warnf("Continuing; you may need to stop it manually before the chart install")
			} else {
				log.Debugf("unit %s not present, skipping stop", unit)
			}
			continue
		}
		if err := osi.DisableService(unit); err != nil {
			if !isMissingUnit(err) {
				log.Warnf("Failed to disable %s: %v", unit, err)
			} else {
				log.Debugf("unit %s not present, skipping disable", unit)
			}
		}
	}
	log.Infof("multipathd stopped and disabled")
	return nil
}

// isMissingUnit reports whether a systemctl error means the unit file does
// not exist on this host (as opposed to a real failure).
func isMissingUnit(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not loaded") ||
		strings.Contains(msg, "does not exist") ||
		strings.Contains(msg, "not found")
}
