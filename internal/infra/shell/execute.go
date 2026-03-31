package shell

import (
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

	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fullCmd := strings.Join(append([]string{command}, args...), " ")
		return fmt.Errorf("command failed: %q: %w", fullCmd, err)
	}

	return nil
}
