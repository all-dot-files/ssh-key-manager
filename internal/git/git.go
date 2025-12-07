package git

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"github.com/all-dot-files/ssh-key-manager/pkg/platform"
)

// Manager manages Git integration
type Manager struct {
	configManager interface {
		GetRepo(path, remote string) (*models.GitRepo, error)
		GetKey(name string) (*models.Key, error)
		GetHost(hostname string) (*models.Host, error)
	}
}

// NewManager creates a new Git manager
func NewManager(configManager interface {
	GetRepo(path, remote string) (*models.GitRepo, error)
	GetKey(name string) (*models.Key, error)
	GetHost(hostname string) (*models.Host, error)
}) *Manager {
	return &Manager{
		configManager: configManager,
	}
}

// BindRepo configures a Git repository to use SKM
func (m *Manager) BindRepo(repoPath, remote, host, user, keyName string) error {
	// Verify repository exists
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository: %s", repoPath)
	}

	// Set Git config for this repository
	// We'll use git config to store SKM metadata
	if err := m.setGitConfig(repoPath, "skm.remote", remote); err != nil {
		return err
	}

	if err := m.setGitConfig(repoPath, "skm.host", host); err != nil {
		return err
	}

	if user != "" {
		if err := m.setGitConfig(repoPath, "skm.user", user); err != nil {
			return err
		}
	}

	if keyName != "" {
		if err := m.setGitConfig(repoPath, "skm.key", keyName); err != nil {
			return err
		}
	}

	return nil
}

// GetRepoConfig gets SKM configuration for a Git repository
func (m *Manager) GetRepoConfig(repoPath string) (*models.GitRepo, error) {
	remote, err := m.getGitConfig(repoPath, "skm.remote")
	if err != nil {
		remote = "origin"
	}

	host, err := m.getGitConfig(repoPath, "skm.host")
	if err != nil {
		return nil, fmt.Errorf("repository not configured with SKM")
	}

	user, _ := m.getGitConfig(repoPath, "skm.user")
	keyName, _ := m.getGitConfig(repoPath, "skm.key")

	return &models.GitRepo{
		Path:    repoPath,
		Remote:  remote,
		Host:    host,
		User:    user,
		KeyName: keyName,
	}, nil
}

// GetSSHCommand returns the GIT_SSH_COMMAND for a repository
func (m *Manager) GetSSHCommand(repoPath string) (string, error) {
	repo, err := m.GetRepoConfig(repoPath)
	if err != nil {
		return "", err
	}

	// Get key
	var keyPath string
	if repo.KeyName != "" {
		key, err := m.configManager.GetKey(repo.KeyName)
		if err != nil {
			return "", fmt.Errorf("failed to get key: %w", err)
		}
		keyPath = key.Path
	} else {
		host, err := m.configManager.GetHost(repo.Host)
		if err != nil {
			return "", fmt.Errorf("failed to get host: %w", err)
		}
		key, err := m.configManager.GetKey(host.KeyName)
		if err != nil {
			return "", fmt.Errorf("failed to get key: %w", err)
		}
		keyPath = key.Path
	}

	// Build SSH command
	return buildSSHCommand(keyPath), nil
}

// WrapCommand wraps a Git command to use the correct SSH key
func (m *Manager) WrapCommand(repoPath string, args []string) error {
	sshCmd, err := m.GetSSHCommand(repoPath)
	if err != nil {
		return err
	}

	// Set GIT_SSH_COMMAND environment variable
	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath
	cmd.Env = append(os.Environ(), fmt.Sprintf("GIT_SSH_COMMAND=%s", sshCmd))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// setGitConfig sets a git config value for a repository
func (m *Manager) setGitConfig(repoPath, key, value string) error {
	cmd := exec.Command("git", "config", "--local", key, value)
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set git config %s: %w", key, err)
	}
	return nil
}

// getGitConfig gets a git config value for a repository
func (m *Manager) getGitConfig(repoPath, key string) (string, error) {
	cmd := exec.Command("git", "config", "--local", "--get", key)
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git config %s: %w", key, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// CreateWrapper creates a wrapper script for Git SSH
func CreateWrapper(wrapperPath, skmPath string) error {
	normalizedSKMPath := filepath.ToSlash(filepath.Clean(skmPath))
	
	// Create platform-specific wrapper(s)
	if err := platform.CreateSSHWrapper(wrapperPath, normalizedSKMPath); err != nil {
		return err
	}

	return nil
}

// InstallGlobalHook installs a global Git hook to intercept operations
func InstallGlobalHook(hooksDir, skmPath string) error {
	hooksDir = filepath.Clean(hooksDir)

	// Create hooks directory
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Create pre-push hook (Unix)
	prePushHook := filepath.Join(hooksDir, "pre-push")
	normalizedSKMPath := filepath.ToSlash(filepath.Clean(skmPath))
	prePushScript := fmt.Sprintf(`#!/bin/sh
# SKM Global Git Hook
# Automatically configures SSH keys for Git operations

%s git auto-config "$PWD"
`, strconv.Quote(normalizedSKMPath))

	if err := os.WriteFile(prePushHook, []byte(prePushScript), 0755); err != nil {
		return fmt.Errorf("failed to create pre-push hook: %w", err)
	}



	// Set core.hooksPath in global git config
	cmd := exec.Command("git", "config", "--global", "core.hooksPath", hooksDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set global hooks path: %w, output: %s", err, string(output))
	}

	return nil
}

// UninstallGlobalHook removes the global Git hook
func UninstallGlobalHook() error {
	cmd := exec.Command("git", "config", "--global", "--unset", "core.hooksPath")
	if err := cmd.Run(); err != nil {
		// It's okay if the config doesn't exist
		return nil
	}
	return nil
}

// AutoConfigureRepo automatically configures a repository based on remote URL
func (m *Manager) AutoConfigureRepo(repoPath string) error {
	// Check if already configured
	if _, err := m.GetRepoConfig(repoPath); err == nil {
		// Already configured, use existing config
		return nil
	}

	// Get remote URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get remote URL: %w", err)
	}

	remoteURL := strings.TrimSpace(string(output))
	if remoteURL == "" {
		return fmt.Errorf("no remote URL found")
	}

	// Parse remote URL to extract host
	host := parseHostFromURL(remoteURL)
	if host == "" {
		return fmt.Errorf("failed to parse host from URL: %s", remoteURL)
	}

	// Check if host exists in config
	hostConfig, err := m.configManager.GetHost(host)
	if err != nil {
		// Host doesn't exist, we'll handle auto-creation later
		return fmt.Errorf("host %s not configured. Run: skm host add %s --user <user> --key <key>", host, host)
	}

	// Bind the repository
	return m.BindRepo(repoPath, "origin", host, "", hostConfig.KeyName)
}

// parseHostFromURL extracts hostname from Git remote URL
func parseHostFromURL(url string) string {
	// Handle SSH URLs: git@github.com:user/repo.git
	if strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, "@")
		if len(parts) > 1 {
			hostParts := strings.Split(parts[1], ":")
			if len(hostParts) > 0 {
				return hostParts[0]
			}
		}
	}

	// Handle HTTPS URLs: https://github.com/user/repo.git
	if strings.HasPrefix(url, "https://") || strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			return parts[0]
		}
	}

	return ""
}

// InstallCredentialHelper installs SKM as a Git credential helper
func (m *Manager) InstallCredentialHelper(repoPath, skmPath string, global bool, hosts, excludes []string) error {
	// Determine config scope
	scope := "--local"
	gitDir := repoPath
	if global {
		scope = "--global"
		gitDir = ""
	}

	// Store host filters in config
	if len(hosts) > 0 {
		hostsStr := strings.Join(hosts, ",")
		if err := runGitConfig(gitDir, scope, "skm.helper.hosts", hostsStr); err != nil {
			return fmt.Errorf("failed to set host filter: %w", err)
		}
	}

	if len(excludes) > 0 {
		excludesStr := strings.Join(excludes, ",")
		if err := runGitConfig(gitDir, scope, "skm.helper.excludes", excludesStr); err != nil {
			return fmt.Errorf("failed to set exclude filter: %w", err)
		}
	}

	// Normalize binary path so core.sshCommand works on Windows/macOS/Linux
	normalizedSKMPath := filepath.ToSlash(filepath.Clean(skmPath))


	// Configure Git to use SKM as credential helper
	// We'll use the core.sshCommand approach which is more reliable for SSH
	wrapperCommand := platform.GetSSHWrapperCommand(normalizedSKMPath)
	wrapperScript := fmt.Sprintf("%s git helper ssh-command", wrapperCommand)

	if global {
		// For global config, we set a wrapper that SKM will intercept
		if err := runGitConfig(gitDir, scope, "skm.helper.enabled", "true"); err != nil {
			return fmt.Errorf("failed to enable helper: %w", err)
		}

		// Set GIT_SSH_COMMAND via environment helper
		if err := runGitConfig(gitDir, scope, "core.sshCommand", wrapperScript); err != nil {
			return fmt.Errorf("failed to set SSH command: %w", err)
		}
	} else {
		// For local config, same approach
		if err := runGitConfig(gitDir, scope, "skm.helper.enabled", "true"); err != nil {
			return fmt.Errorf("failed to enable helper: %w", err)
		}

		if err := runGitConfig(gitDir, scope, "core.sshCommand", wrapperScript); err != nil {
			return fmt.Errorf("failed to set SSH command: %w", err)
		}
	}
	
	return nil
}

// UninstallCredentialHelper removes SKM as a Git credential helper
func (m *Manager) UninstallCredentialHelper(repoPath string, global bool) error {
	scope := "--local"
	gitDir := repoPath
	if global {
		scope = "--global"
		gitDir = ""
	}

	// Remove all SKM-related config
	runGitConfig(gitDir, scope, "--unset", "skm.helper.enabled")
	runGitConfig(gitDir, scope, "--unset", "skm.helper.hosts")
	runGitConfig(gitDir, scope, "--unset", "skm.helper.excludes")
	runGitConfig(gitDir, scope, "--unset", "core.sshCommand")

	return nil
}

// CredentialHelperGet handles the 'get' operation of the credential helper protocol
func (m *Manager) CredentialHelperGet() error {
	// Read input from stdin (credential helper protocol)
	scanner := bufio.NewScanner(os.Stdin)
	wants := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}

		key, value := parts[0], parts[1]
		wants[key] = value
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	// Get protocol and host from attributes
	protocol := wants["protocol"]
	host := wants["host"]

	if host == "" {
		// Not a credential we can help with
		return nil
	}

	// Only handle SSH protocol (git@host format uses empty protocol, SSH URLs use "ssh")
	// For HTTPS, we skip (unless we want to handle HTTPS git operations)
	if protocol == "https" || protocol == "http" {
		return nil
	}

	// Check if we should handle this host
	if !m.shouldHandleHost(host) {
		return nil
	}

	// Get the host configuration
	hostConfig, err := m.configManager.GetHost(host)
	if err != nil {
		// Host not configured, silently skip
		return nil
	}

	// Get the key
	key, err := m.configManager.GetKey(hostConfig.KeyName)
	if err != nil {
		// Key not found, silently skip
		return nil
	}

	// Check if a specific username was requested
	wantedUsername := wants["username"]
	gotUsername := hostConfig.User
	if gotUsername == "" {
		gotUsername = "git" // Default for most git hosts
	}

	// If a username was specified and doesn't match, skip
	if wantedUsername != "" && wantedUsername != gotUsername {
		return nil
	}

	// Output credentials in the format Git expects
	// For SSH authentication, we output the username and use the key path as "password"
	// Git won't use this directly, but our SSH wrapper will intercept it
	if protocol != "" {
		fmt.Printf("protocol=%s\n", protocol)
	}
	fmt.Printf("host=%s\n", host)
	fmt.Printf("username=%s\n", gotUsername)

	// Store key path as metadata that our SSH wrapper can use
	// We use a special marker that our wrapper will recognize
	fmt.Printf("password=skm-key:%s\n", key.Path)

	return nil
}

// CredentialHelperStore handles the 'store' operation of the credential helper protocol
// We pretend to implement "store" but do nothing since we manage credentials via SKM config
func (m *Manager) CredentialHelperStore() error {
	// Read and discard input to satisfy the protocol
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
	}
	// Silently succeed - we don't need to store anything
	return nil
}

// CredentialHelperErase handles the 'erase' operation of the credential helper protocol
// We pretend to implement "erase" but do nothing since we don't want Git to cause logout
func (m *Manager) CredentialHelperErase() error {
	// Read and discard input to satisfy the protocol
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		}
	}
	// Silently succeed - we don't erase anything
	return nil
}

// shouldHandleHost checks if the helper should handle the given host
func (m *Manager) shouldHandleHost(host string) bool {
	// Try to get host/exclude filters from git config
	// Optimize: Get all skm.helper.* config in one go if possible, or just cache it.
	// For now, we'll just read them.

	// Check current directory for git config
	// We can use "git config --get-regexp skm.helper" to get all relevant keys
	
	hostsConfig, _ := getGitConfigValue("", "skm.helper.hosts")
	excludesConfig, _ := getGitConfigValue("", "skm.helper.excludes")

	// Parse host filters
	if hostsConfig != "" {
		allowedHosts := strings.Split(hostsConfig, ",")
		found := false
		for _, h := range allowedHosts {
			if strings.TrimSpace(h) == host {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Parse exclude filters
	if excludesConfig != "" {
		excludedHosts := strings.Split(excludesConfig, ",")
		for _, h := range excludedHosts {
			if strings.TrimSpace(h) == host {
				return false
			}
		}
	}

	return true
}

// runGitConfig runs a git config command
func runGitConfig(gitDir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"config"}, args...)...)
	if gitDir != "" {
		cmd.Dir = gitDir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git config failed: %w, output: %s", err, string(output))
	}
	return nil
}

// getGitConfigValue gets a git config value
func getGitConfigValue(gitDir, key string) (string, error) {
	cmd := exec.Command("git", "config", "--get", key)
	if gitDir != "" {
		cmd.Dir = gitDir
	}
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetSSHCommandForHost returns the SSH command to use for a specific host
func (m *Manager) GetSSHCommandForHost(host string) (string, error) {
	// Check if we should handle this host
	if !m.shouldHandleHost(host) {
		return "", fmt.Errorf("host not configured for SKM: %s", host)
	}

	// Get the host configuration
	hostConfig, err := m.configManager.GetHost(host)
	if err != nil {
		return "", fmt.Errorf("host not found: %w", err)
	}

	// Get the key
	key, err := m.configManager.GetKey(hostConfig.KeyName)
	if err != nil {
		return "", fmt.Errorf("key not found: %w", err)
	}

	// Build SSH command
	knownHosts := filepath.ToSlash(os.DevNull)
	sshCmd := buildSSHCommand(
		key.Path,
		"-o", fmt.Sprintf("UserKnownHostsFile=%s", knownHosts),
		"-o", "StrictHostKeyChecking=no",
	)
	return sshCmd, nil
}

// HandleSSHCommand handles SSH command wrapping for Git operations
func (m *Manager) HandleSSHCommand(args []string) error {
	// Git calls SSH with arguments like: user@host -p port command
	// We need to extract the host and inject the correct SSH key

	if len(args) == 0 {
		// No arguments, just execute SSH
		return execSSH(args)
	}

	// Find the host argument (usually first arg, format: user@host or just host)
	hostArg := ""
	for i, arg := range args {
		// Skip flags
		if strings.HasPrefix(arg, "-") {
			continue
		}
		// First non-flag argument should be the host
		hostArg = args[i]
		break
	}

	if hostArg == "" {
		// No host found, just execute SSH normally
		return execSSH(args)
	}

	// Extract host from user@host format
	host := hostArg
	if strings.Contains(hostArg, "@") {
		parts := strings.Split(hostArg, "@")
		if len(parts) == 2 {
			host = parts[1]
		}
	}

	// Remove port if present (host:port)
	if strings.Contains(host, ":") {
		host = strings.Split(host, ":")[0]
	}

	// Check if we should handle this host
	if !m.shouldHandleHost(host) {
		return execSSH(args)
	}

	// Get the host configuration
	hostConfig, err := m.configManager.GetHost(host)
	if err != nil {
		// Host not configured, use default SSH
		return execSSH(args)
	}

	// Get the key
	key, err := m.configManager.GetKey(hostConfig.KeyName)
	if err != nil {
		return fmt.Errorf("failed to get key: %w", err)
	}

	// Build SSH command with the correct key
	sshArgs := []string{
		"-i", key.Path,
		"-o", "IdentitiesOnly=yes",
	}

	// Append original arguments
	sshArgs = append(sshArgs, args...)

	return execSSH(sshArgs)
}

func buildSSHCommand(keyPath string, extraArgs ...string) string {
	normalizedKey := filepath.ToSlash(filepath.Clean(keyPath))
	quotedKey := strconv.Quote(normalizedKey)

	args := []string{"ssh", "-i", quotedKey, "-o", "IdentitiesOnly=yes"}
	args = append(args, extraArgs...)

	return strings.Join(args, " ")
}

// execSSH executes the SSH command with the given arguments
func execSSH(args []string) error {
	cmd := exec.Command("ssh", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
