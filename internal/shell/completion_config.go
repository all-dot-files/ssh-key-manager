package shell

import (
	"fmt"
	"os"
	"path/filepath"
)

// ShellType represents a supported shell.
type ShellType string

const (
	ShellZsh  ShellType = "zsh"
	ShellBash ShellType = "bash"
	ShellFish ShellType = "fish"
)

// CompletionConfig holds shell-specific completion settings.
type CompletionConfig struct {
	Shell       ShellType
	InstallPath string
	ProfilePath string
}

// DefaultInstallPath returns a best-effort install path for the given shell.
func DefaultInstallPath(shell ShellType) string {
	switch shell {
	case ShellZsh:
		return filepath.Join(os.Getenv("HOME"), ".oh-my-zsh", "completions", "_skm")
	case ShellBash:
		return "/etc/bash_completion.d/skm"
	case ShellFish:
		return filepath.Join(os.Getenv("HOME"), ".config", "fish", "completions", "skm.fish")
	default:
		return ""
	}
}

// ValidateWritable checks if the path is writable; returns a helpful error if not.
func ValidateWritable(path string) error {
	dir := filepath.Dir(path)
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("completion path not accessible (%s): %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("completion path parent is not a directory: %s", dir)
	}
	testFile := filepath.Join(dir, ".skm-completion-writecheck")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("cannot write completion file to %s: %w", dir, err)
	}
	f.Close()
	_ = os.Remove(testFile)
	return nil
}
