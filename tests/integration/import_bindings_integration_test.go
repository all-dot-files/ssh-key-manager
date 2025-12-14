package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
	"github.com/all-dot-files/ssh-key-manager/internal/config"
)

func TestImportPlannerApplyBindings(t *testing.T) {
	dir := t.TempDir()
	priv := filepath.Join(dir, "id_ed25519")
	pub := priv + ".pub"
	if err := os.WriteFile(priv, []byte("PRIVATE"), 0600); err != nil {
		t.Fatalf("write priv: %v", err)
	}
	if err := os.WriteFile(pub, []byte(cli.GenerateTestPubForTest()), 0644); err != nil {
		t.Fatalf("write pub: %v", err)
	}
	configContent := "Host testhost\n  User git\n  IdentityFile " + priv + "\n"
	if err := os.WriteFile(filepath.Join(dir, "config"), []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	mgr, _ := config.NewManager(cfgPath)
	_ = mgr.Load()

	planner := cli.NewImportPlanner(mgr, dir, "", false)
	planner.Keys = []cli.ImportedKey{{
		Alias:       "id_ed25519",
		PrivatePath: priv,
		PublicPath:  pub,
		Fingerprint: "fp",
		Status:      "new",
	}}
	planner.KeyMapForTest(map[string]cli.ImportedKey{
		filepath.Clean(priv): {Alias: "id_ed25519"},
	})

	if _, err := planner.DetectBindings(); err != nil {
		t.Fatalf("detect bindings: %v", err)
	}

	if err := planner.Apply(); err != nil {
		t.Fatalf("apply: %v", err)
	}

	keys, err := mgr.ListKeys()
	if err != nil {
		t.Fatalf("list keys: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	hosts, err := mgr.ListHosts()
	if err != nil {
		t.Fatalf("list hosts: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].KeyName != "id_ed25519" {
		t.Fatalf("expected host bound to id_ed25519, got %s", hosts[0].KeyName)
	}
}
