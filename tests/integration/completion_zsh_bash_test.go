package integration

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
	"github.com/all-dot-files/ssh-key-manager/internal/config"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

func setupCompletionManager(t *testing.T) {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	manager, err := config.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if err := manager.Load(); err != nil {
		t.Fatalf("load manager: %v", err)
	}
	cfg := manager.Get()
	cfg.Keys = []models.Key{
		{Name: "work"},
		{Name: "personal"},
	}
	if err := manager.Save(); err != nil {
		t.Fatalf("save manager: %v", err)
	}
	cli.ConfigManagerForTest(manager)
}

func TestBashCompletionGeneration(t *testing.T) {
	setupCompletionManager(t)
	var out bytes.Buffer
	cmd := cli.CompletionCommandForTest()
	cmd.Flags().Set("out", "")
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.RunE(cmd, []string{"bash"}); err != nil {
		t.Fatalf("bash completion generation failed: %v", err)
	}
	if out.Len() == 0 {
		t.Fatalf("expected completion output, got none")
	}
}

func TestFishCompletionGeneration(t *testing.T) {
	setupCompletionManager(t)
	var out bytes.Buffer
	cmd := cli.CompletionCommandForTest()
	cmd.Flags().Set("out", "")
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	if err := cmd.RunE(cmd, []string{"fish"}); err != nil {
		t.Fatalf("fish completion generation failed: %v", err)
	}
	if out.Len() == 0 {
		t.Fatalf("expected completion output, got none")
	}
}
