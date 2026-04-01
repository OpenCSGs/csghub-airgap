package user

import (
	"fmt"
	"os"
)

// EnsureRoot returns error if current user is not root
func EnsureRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must be run as root user (UID 0)")
	}
	return nil
}
