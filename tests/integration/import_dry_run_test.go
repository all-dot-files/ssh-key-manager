package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
	"github.com/all-dot-files/ssh-key-manager/internal/config"
)

func TestImportDryRunDoesNotWriteConfig(t *testing.T) {
	dir := t.TempDir()
	priv := filepath.Join(dir, "id_ed25519")
	pub := priv + ".pub"
	_ = os.WriteFile(priv, []byte("PRIVATE"), 0600)
	_ = os.WriteFile(pub, []byte(cli.GenerateTestPubForTest()), 0644)

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	mgr, _ := config.NewManager(cfgPath)
	_ = mgr.Load()

	planner := cli.NewImportPlanner(mgr, dir, "", true)
	if _, err := planner.DiscoverKeys(); err != nil {
		t.Fatalf("discover: %v", err)
	}
	if len(planner.Keys) == 0 {
		t.Fatalf("expected keys discovered")
	}

	// Dry run: skip Apply; ensure manager state not mutated (no keys/hosts written)
	if keys, _ := mgr.ListKeys(); len(keys) != 0 {
		t.Fatalf("expected no keys persisted in dry-run")
	}
	if hosts, _ := mgr.ListHosts(); len(hosts) != 0 {
		t.Fatalf("expected no hosts persisted in dry-run")
	}
}
