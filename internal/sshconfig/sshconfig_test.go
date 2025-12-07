package sshconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	if manager.configPath != filepath.Join(tmpDir, "config") {
		t.Errorf("expected config path %s, got %s", filepath.Join(tmpDir, "config"), manager.configPath)
	}
}

func TestUpdateConfig(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create dummy keys
	keys := map[string]*models.Key{
		"testkey": {
			Name: "testkey",
			Path: "/path/to/key",
		},
	}

	// Create dummy hosts
	hosts := []models.Host{
		{
			Host:    "example.com",
			User:    "git",
			KeyName: "testkey",
			Port:    2222,
		},
	}

	// Test creating new config
	err := manager.UpdateConfig(hosts, keys)
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(manager.configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}

	strContent := string(content)
	if !strings.Contains(strContent, "Host example.com") {
		t.Error("config missing Host entry")
	}
	if !strings.Contains(strContent, "IdentityFile /path/to/key") {
		t.Error("config missing IdentityFile")
	}
	if !strings.Contains(strContent, "Port 2222") {
		t.Error("config missing Port")
	}
	if !strings.Contains(strContent, skmManagedStart) {
		t.Error("config missing managed start marker")
	}

	// Test updating existing config
	// Add some unmanaged content
	unmanaged := "Host other\n  User other\n"
	err = os.WriteFile(manager.configPath, []byte(unmanaged+strContent), 0600)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Update again
	err = manager.UpdateConfig(hosts, keys)
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	// Verify content again
	content, err = os.ReadFile(manager.configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	strContent = string(content)

	if !strings.Contains(strContent, "Host other") {
		t.Error("lost unmanaged content")
	}
	if strings.Count(strContent, skmManagedStart) != 1 {
		t.Error("duplicate managed section")
	}
}

func TestRemoveManagedSection(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewManager(tmpDir)

	// Create config with managed section
	content := `Host other
  User other

# === SKM MANAGED START ===
Host example.com
  User git
# === SKM MANAGED END ===
`
	err := os.WriteFile(manager.configPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Remove section
	err = manager.RemoveManagedSection()
	if err != nil {
		t.Fatalf("RemoveManagedSection failed: %v", err)
	}

	// Verify
	newContent, err := os.ReadFile(manager.configPath)
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	strContent := string(newContent)

	if strings.Contains(strContent, skmManagedStart) {
		t.Error("managed section not removed")
	}
	if !strings.Contains(strContent, "Host other") {
		t.Error("unmanaged content removed")
	}
}
