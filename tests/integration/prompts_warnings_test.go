package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
)

func TestCompletionFailsOnUnwritablePath(t *testing.T) {
	dir := t.TempDir()
	denyDir := filepath.Join(dir, "deny")
	if err := os.MkdirAll(denyDir, 0o500); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	target := filepath.Join(denyDir, "skm.fish")

	cmd := cli.CompletionCommandForTest()
	cmd.Flags().Set("out", target)
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	if err := cmd.RunE(cmd, []string{"fish"}); err == nil {
		t.Fatalf("expected failure when writing to unwritable path")
	}
	cmd.Flags().Set("out", "")
}
