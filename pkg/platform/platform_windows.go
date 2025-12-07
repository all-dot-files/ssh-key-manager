//go:build windows

package platform

import (
	"fmt"
	"os"
	"strconv"
)

// CreateSSHWrapper creates the Git SSH wrapper script for Windows (batch + shell)
func CreateSSHWrapper(wrapperPath, skmPath string) error {
	// Create shell script (for Git Bash / MSYS2)
	script := fmt.Sprintf(`#!/bin/sh
# SKM Git SSH Wrapper
# This wrapper ensures Git operations use the correct SSH key

exec %s ssh-wrapper "$@"
`, strconv.Quote(skmPath))

	if err := os.WriteFile(wrapperPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write wrapper script: %w", err)
	}

	// Create batch script (for cmd.exe)
	batchScript := fmt.Sprintf(`@echo off
REM SKM Git SSH Wrapper
"%s" ssh-wrapper %%*
`, skmPath)

	batchPath := wrapperPath + ".cmd"
	if err := os.WriteFile(batchPath, []byte(batchScript), 0755); err != nil {
		return fmt.Errorf("failed to write wrapper batch script: %w", err)
	}

	return nil
}

// GetSSHWrapperCommand returns the command to use for core.sshCommand
func GetSSHWrapperCommand(wrapperPath string) string {
	// On Windows, point to the .cmd file to ensure it runs correctly in all shells
	return strconv.Quote(wrapperPath + ".cmd")
}
