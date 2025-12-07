//go:build !windows

package platform

import (
	"fmt"
	"os"
	"strconv"
)

// CreateSSHWrapper creates the Git SSH wrapper script for Unix
func CreateSSHWrapper(wrapperPath, skmPath string) error {
	script := fmt.Sprintf(`#!/bin/sh
# SKM Git SSH Wrapper
# This wrapper ensures Git operations use the correct SSH key

exec %s ssh-wrapper "$@"
`, strconv.Quote(skmPath))

	if err := os.WriteFile(wrapperPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}

	return nil
}

// GetSSHWrapperCommand returns the command to use for core.sshCommand
func GetSSHWrapperCommand(wrapperPath string) string {
	// On Unix, we just use the script path directly
	return strconv.Quote(wrapperPath)
}
