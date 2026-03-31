package user

import (
	"errors"
	"os"
)

// EnsureRoot returns error if current user is not root
func EnsureRoot() error {
	if os.Geteuid() != 0 {
		return errors.New("operation must be run as root user (UID 0)")
	}
	return nil
}
