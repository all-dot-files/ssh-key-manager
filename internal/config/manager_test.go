package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

func TestInitialize(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	manager, err := NewManager(configPath)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Test Initialize with custom SSH dir
	sshDir := filepath.Join(tmpDir, "ssh")
	err = manager.Initialize("test-device", sshDir)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	cfg := manager.Get()
	if cfg.DeviceName != "test-device" {
		t.Errorf("expected device name test-device, got %s", cfg.DeviceName)
	}
	if cfg.SSHDir != sshDir {
		t.Errorf("expected ssh dir %s, got %s", sshDir, cfg.SSHDir)
	}
	if cfg.KeystorePath == "" {
		t.Error("keystore path is empty")
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file not created")
	}
}

func TestKeyManagement(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	manager, _ := NewManager(configPath)
	manager.Initialize("test", "")

	key := models.Key{
		Name: "testkey",
		Type: models.KeyTypeED25519,
		Path: "/tmp/key",
	}

	// Add Key
	err := manager.AddKey(key)
	if err != nil {
		t.Fatalf("AddKey failed: %v", err)
	}

	// Get Key
	retrieved, err := manager.GetKey("testkey")
	if err != nil {
		t.Fatalf("GetKey failed: %v", err)
	}
	if retrieved.Path != "/tmp/key" {
		t.Error("retrieved key has wrong path")
	}

	// Remove Key
	err = manager.RemoveKey("testkey")
	if err != nil {
		t.Fatalf("RemoveKey failed: %v", err)
	}

	_, err = manager.GetKey("testkey")
	if err == nil {
		t.Error("key should have been removed")
	}
}
