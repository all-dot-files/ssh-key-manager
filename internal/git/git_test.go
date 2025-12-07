package git

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCreateWrapper(t *testing.T) {
	tmpDir := t.TempDir()
	wrapperPath := filepath.Join(tmpDir, "ssh-wrapper")
	skmPath := "/usr/local/bin/skm"

	err := CreateWrapper(wrapperPath, skmPath)
	if err != nil {
		t.Fatalf("CreateWrapper failed: %v", err)
	}

	// Check Unix wrapper
	content, err := os.ReadFile(wrapperPath)
	if err != nil {
		t.Fatalf("failed to read wrapper: %v", err)
	}
	strContent := string(content)
	if !strings.Contains(strContent, "/usr/local/bin/skm") {
		t.Error("wrapper missing skm path")
	}

	// Check Windows wrapper if on Windows
	if runtime.GOOS == "windows" {
		batchPath := wrapperPath + ".cmd"
		content, err := os.ReadFile(batchPath)
		if err != nil {
			t.Fatalf("failed to read batch wrapper: %v", err)
		}
		strContent = string(content)
		if !strings.Contains(strContent, "skm") {
			t.Error("batch wrapper missing skm path")
		}
	}
}

// We can't easily test shouldHandleHost without mocking git config commands or setting up a real git repo.
// For now, we'll skip complex git integration tests that require external commands.
