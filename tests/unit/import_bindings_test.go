package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/cli"
)

func TestDetectBindingsMatchesKeyByIdentityFile(t *testing.T) {
	dir := t.TempDir()
	priv := filepath.Join(dir, "id_ed25519")
	pub := priv + ".pub"
	if err := os.WriteFile(priv, []byte("PRIVATE"), 0600); err != nil {
		t.Fatalf("write priv: %v", err)
	}
	if err := os.WriteFile(pub, []byte(cli.GenerateTestPubForTest()), 0644); err != nil {
		t.Fatalf("write pub: %v", err)
	}
	cfg := filepath.Join(dir, "config")
	content := "Host testhost\n  User git\n  IdentityFile " + priv + "\n"
	if err := os.WriteFile(cfg, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	planner := cli.NewImportPlanner(nil, dir, "", true)
	planner.Keys = []cli.ImportedKey{{Alias: "id_ed25519", PrivatePath: priv, PublicPath: pub}}
	planner.Bindings = nil
	planner.Warnings()
	planner.KeyMapForTest(map[string]cli.ImportedKey{
		filepath.Clean(priv): {Alias: "id_ed25519"},
	})

	bindings, err := planner.DetectBindings()
	if err != nil {
		t.Fatalf("detect bindings: %v", err)
	}
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	if bindings[0].TargetAlias != "id_ed25519" {
		t.Fatalf("expected alias id_ed25519, got %s", bindings[0].TargetAlias)
	}
}
