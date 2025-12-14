package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
	"github.com/all-dot-files/ssh-key-manager/internal/config"
)

func TestImportWarnsOnMissingPublicKey(t *testing.T) {
	dir := t.TempDir()
	priv := filepath.Join(dir, "id_ed25519")
	_ = os.WriteFile(priv, []byte("PRIVATE"), 0600)
	// no public key file

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	mgr, _ := config.NewManager(cfgPath)
	_ = mgr.Load()

	planner := cli.NewImportPlanner(mgr, dir, "", false)
	keys, err := planner.DiscoverKeys()
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("expected zero keys due to missing pub, got %d", len(keys))
	}
	if len(planner.Warnings()) == 0 {
		t.Fatalf("expected warnings for missing public key")
	}
}
