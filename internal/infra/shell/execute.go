package shell

import (
	"csghub-airgap/internal/config"
	"csghub-airgap/internal/log"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Execute executes a system command with optional arguments.
// It returns the combined standard output, standard error, and any execution error.
func Execute(command string, args ...string) error {
	if command == "" {
		return errors.New("command must not be empty")
	}

	if config.DryRun {
		// Strip the bash -c wrapper so the dry-run prints the real command.
		if command == "bash" || command == "/bin/bash" {
			if len(args) > 1 && args[0] == "-c" {
				log.Infof("[DRY-RUN] $ %s", strings.Join(args[1:], " "))
				return nil
			}
		}
		log.Infof("[DRY-RUN] $ %s %s", command, strings.Join(args, " "))
		return nil
	}

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fullCmd := strings.Join(append([]string{command}, args...), " ")
		return fmt.Errorf("command failed: %q: %v", fullCmd, err)
	}

	return nil
}
