package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/all-dot-files/ssh-key-manager/internal/config"
	"github.com/all-dot-files/ssh-key-manager/internal/git"
	"github.com/all-dot-files/ssh-key-manager/internal/keystore"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"github.com/all-dot-files/ssh-key-manager/internal/sshconfig"
)

// tempDir creates an isolated path under tests/.tmp to keep artifacts in-repo.
func tempDir(t *testing.T, base, prefix string) string {
	t.Helper()

	baseAbs, err := filepath.Abs(base)
	if err != nil {
		t.Fatalf("failed to resolve base path: %v", err)
	}

	if err := os.MkdirAll(baseAbs, 0755); err != nil {
		t.Fatalf("failed to create base temp dir: %v", err)
	}

	dir, err := os.MkdirTemp(baseAbs, prefix)
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// newIsolatedConfig boots a fresh config/keystore/ssh dir using a temporary HOME.
func newIsolatedConfig(t *testing.T) (*config.Manager, *keystore.KeyStore, string) {
	t.Helper()

	base := filepath.Join("tests", ".tmp")
	home := tempDir(t, base, "home-")
	t.Setenv("HOME", home)

	cfgManager, err := config.NewManager("")
	if err != nil {
		t.Fatalf("failed to create config manager: %v", err)
	}

	if err := cfgManager.Initialize("test-device", ""); err != nil {
		t.Fatalf("failed to initialize config: %v", err)
	}

	cfg := cfgManager.Get()
	ks, err := keystore.NewKeyStore(cfg.KeystorePath)
	if err != nil {
		t.Fatalf("failed to create keystore: %v", err)
	}

	return cfgManager, ks, home
}

func TestReadmeQuickstartWorkflow(t *testing.T) {
	cfgManager, ks, home := newIsolatedConfig(t)
	cfg := cfgManager.Get()

	// Step 2: generate an ED25519 key and record it in config.
	workKey, err := ks.GenerateKey("work", models.KeyTypeED25519, "", 0)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}
	if err := cfgManager.AddKey(*workKey); err != nil {
		t.Fatalf("failed to add key to config: %v", err)
	}

	// Step 3: add a host and write SSH config.
	host := models.Host{
		Host:    "github.com",
		User:    "git",
		KeyName: workKey.Name,
	}
	if err := cfgManager.AddHost(host); err != nil {
		t.Fatalf("failed to add host: %v", err)
	}

	sshMgr := sshconfig.NewManager(cfg.SSHDir)
	keyMap := map[string]*models.Key{workKey.Name: workKey}
    
    // Refresh config to get latest hosts
    cfg = cfgManager.Get()
	if err := sshMgr.UpdateConfig(cfg.Hosts, keyMap); err != nil {
		t.Fatalf("failed to update ssh config: %v", err)
	}

	sshConfigPath := filepath.Join(cfg.SSHDir, "config")
	sshCfg, err := os.ReadFile(sshConfigPath)
	if err != nil {
		t.Fatalf("failed to read ssh config: %v", err)
	}
	if !strings.Contains(string(sshCfg), "Host github.com") {
		t.Fatalf("ssh config missing host block: %s", sshCfg)
	}
	if !strings.Contains(string(sshCfg), workKey.Path) {
		t.Fatalf("ssh config missing key path: %s", sshCfg)
	}

	// Step 4: bind a git repo and ensure SKM metadata and SSH command are set.
	repoBase := filepath.Join("tests", ".tmp")
	repoDir := tempDir(t, repoBase, "repo-")
	initCmd := exec.Command("git", "init")
	initCmd.Dir = repoDir
	initCmd.Env = append(os.Environ(), "HOME="+home)
	if out, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}

	gitMgr := git.NewManager(cfgManager)
	if err := gitMgr.BindRepo(repoDir, "origin", host.Host, host.User, workKey.Name); err != nil {
		t.Fatalf("failed to bind repo: %v", err)
	}

	sshCmd, err := gitMgr.GetSSHCommand(repoDir)
	if err != nil {
		t.Fatalf("failed to get ssh command: %v", err)
	}
	if !strings.Contains(sshCmd, filepath.ToSlash(workKey.Path)) {
		t.Fatalf("ssh command missing key path: %s", sshCmd)
	}

	if got := gitConfigGet(t, repoDir, "skm.host"); got != host.Host {
		t.Fatalf("expected skm.host=%s, got %s", host.Host, got)
	}
	if got := gitConfigGet(t, repoDir, "skm.remote"); got != "origin" {
		t.Fatalf("expected skm.remote=origin, got %s", got)
	}
}

func TestReadmeKeyTypesAndPassphrase(t *testing.T) {
	cfgManager, ks, _ := newIsolatedConfig(t)

	edKey, err := ks.GenerateKey("ed25519-key", models.KeyTypeED25519, "", 0)
	if err != nil {
		t.Fatalf("failed to generate ed25519 key: %v", err)
	}
	rsaKey, err := ks.GenerateKey("rsa-protected", models.KeyTypeRSA, "s3cr3t-pass", 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	if err := cfgManager.AddKey(*edKey); err != nil {
		t.Fatalf("failed to add ed25519 key: %v", err)
	}
	if err := cfgManager.AddKey(*rsaKey); err != nil {
		t.Fatalf("failed to add rsa key: %v", err)
	}

	if rsaKey.HasPassphrase != true {
		t.Fatalf("expected rsa key to be passphrase protected")
	}
	if _, err := os.Stat(edKey.Path); err != nil {
		t.Fatalf("ed25519 private key missing: %v", err)
	}
	if _, err := os.Stat(rsaKey.Path); err != nil {
		t.Fatalf("rsa private key missing: %v", err)
	}

	// Load and decrypt the RSA key to confirm passphrase handling.
	plaintext, err := ks.LoadPrivateKey(rsaKey, "s3cr3t-pass")
	if err != nil {
		t.Fatalf("failed to decrypt rsa key: %v", err)
	}
	if !strings.Contains(string(plaintext), "PRIVATE KEY") {
		t.Fatalf("decrypted rsa key appears invalid")
	}

	// Persisted config should still be writable with multiple keys present.
	if err := cfgManager.Save(); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	if cfg := cfgManager.Get(); len(cfg.Keys) < 2 {
		t.Fatalf("config missing generated keys")
	}
}

func gitConfigGet(t *testing.T, repoPath, key string) string {
	t.Helper()

	cmd := exec.Command("git", "config", "--local", "--get", key)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git config --get %s failed: %v", key, err)
	}
	return strings.TrimSpace(string(output))
}

func TestGitHookInstallCreatesPrePushAndConfig(t *testing.T) {
	cfgManager, _, home := newIsolatedConfig(t)
	cfg := cfgManager.Get()

	if err := os.MkdirAll(cfg.SSHDir, 0700); err != nil {
		t.Fatalf("failed to create ssh dir: %v", err)
	}

	// Prepare an empty global git config file under the isolated HOME.
	gitconfig := filepath.Join(home, ".gitconfig")
	if err := os.WriteFile(gitconfig, []byte{}, 0644); err != nil {
		t.Fatalf("failed to pre-create gitconfig: %v", err)
	}
	xdgConfig := filepath.Join(home, ".config")
	if err := os.MkdirAll(xdgConfig, 0700); err != nil {
		t.Fatalf("failed to create xdg config dir: %v", err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", gitconfig)
	t.Setenv("XDG_CONFIG_HOME", xdgConfig)

	hooksDir := filepath.Clean(filepath.Join(cfg.SSHDir, "..", ".git-hooks"))
	skmPath, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get executable path: %v", err)
	}

	if err := git.InstallGlobalHook(hooksDir, skmPath); err != nil {
		t.Fatalf("failed to install global hook: %v", err)
	}

	prePush := filepath.Join(hooksDir, "pre-push")
	info, err := os.Stat(prePush)
	if err != nil {
		t.Fatalf("pre-push hook missing: %v", err)
	}
	if info.Mode()&0100 == 0 {
		t.Fatalf("pre-push hook is not executable")
	}

	content, err := os.ReadFile(prePush)
	if err != nil {
		t.Fatalf("failed to read pre-push: %v", err)
	}
	if !strings.Contains(string(content), "git auto-config") {
		t.Fatalf("pre-push hook missing auto-config call:\n%s", content)
	}

	coreHooksPath := gitConfigGlobalGet(t, home, "core.hooksPath")
	if filepath.Clean(coreHooksPath) != hooksDir {
		t.Fatalf("core.hooksPath expected %s, got %s", hooksDir, coreHooksPath)
	}
}

func gitConfigGlobalGet(t *testing.T, home, key string) string {
	t.Helper()

	cmd := exec.Command("git", "config", "--global", "--get", key)
	cmd.Env = append(os.Environ(), "HOME="+home)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("git config --global --get %s failed: %v", key, err)
	}
	return strings.TrimSpace(string(output))
}
