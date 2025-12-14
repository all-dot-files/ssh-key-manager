package unit

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
	"github.com/all-dot-files/ssh-key-manager/internal/config"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

func TestDiscoverKeysMarksConflicts(t *testing.T) {
	dir := t.TempDir()
	priv := filepath.Join(dir, "id_ed25519")
	pub := priv + ".pub"
	if err := os.WriteFile(priv, []byte("PRIVATE"), 0600); err != nil {
		t.Fatalf("write priv: %v", err)
	}
	if err := os.WriteFile(pub, []byte(cli.GenerateTestPubForTest()), 0644); err != nil {
		t.Fatalf("write pub: %v", err)
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	mgr, _ := config.NewManager(cfgPath)
	_ = mgr.Load()
	_ = mgr.AddKey(models.Key{
		Name:      "id_ed25519",
		Path:      priv,
		PubPath:   pub,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	planner := cli.NewImportPlanner(mgr, dir, "", false)
	keys, err := planner.DiscoverKeys()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Status != "conflict" {
		t.Fatalf("expected conflict, got %s", keys[0].Status)
	}
}
