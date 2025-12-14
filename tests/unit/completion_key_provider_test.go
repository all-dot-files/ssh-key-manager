package unit

import (
	"path/filepath"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
	"github.com/all-dot-files/ssh-key-manager/internal/config"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

func TestKeyCompletionProviderFallback(t *testing.T) {
	// nil manager should return no names
	provider := cli.NewKeyCompletionProviderForTest(nil, nil)
	if names := provider.Names(); len(names) != 0 {
		t.Fatalf("expected empty names for nil manager, got %v", names)
	}
}

func TestKeyCompletionProviderListsNames(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	mgr, err := config.NewManager(cfgPath)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if err := mgr.Load(); err != nil {
		t.Fatalf("load manager: %v", err)
	}
	cfg := mgr.Get()
	cfg.Keys = []models.Key{{Name: "work"}, {Name: "personal"}}
	if err := mgr.Save(); err != nil {
		t.Fatalf("save manager: %v", err)
	}

	provider := cli.NewKeyCompletionProviderForTest(mgr, nil)
	names := provider.Names()
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %v", names)
	}
}
