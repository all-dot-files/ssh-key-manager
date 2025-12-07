package sshconfig

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

const (
	skmManagedStart = "# === SKM MANAGED START ==="
	skmManagedEnd   = "# === SKM MANAGED END ==="
)

// Manager manages SSH config file
type Manager struct {
	configPath string
}

// NewManager creates a new SSH config manager
func NewManager(sshDir string) *Manager {
	return &Manager{
		configPath: filepath.Join(sshDir, "config"),
	}
}

// UpdateConfig updates the SSH config file with managed hosts
func (m *Manager) UpdateConfig(hosts []models.Host, keys map[string]*models.Key) error {
	// Read existing config
	existingContent := ""
	managedContent := []string{}
	inManagedSection := false

	if _, err := os.Stat(m.configPath); err == nil {
		file, err := os.Open(m.configPath)
		if err != nil {
			return fmt.Errorf("failed to open SSH config: %w", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			if strings.Contains(line, skmManagedStart) {
				inManagedSection = true
				continue
			}

			if strings.Contains(line, skmManagedEnd) {
				inManagedSection = false
				continue
			}

			if !inManagedSection {
				existingContent += line + "\n"
			}
		}

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to read SSH config: %w", err)
		}
	}

	// Generate managed section
	managedContent = append(managedContent, skmManagedStart)
	managedContent = append(managedContent, "# This section is managed by SKM. Do not edit manually.")
	managedContent = append(managedContent, "")

	for _, host := range hosts {
		key, ok := keys[host.KeyName]
		if !ok {
			continue
		}

		managedContent = append(managedContent, fmt.Sprintf("Host %s", host.Host))
		if host.Hostname != "" {
			managedContent = append(managedContent, fmt.Sprintf("    HostName %s", host.Hostname))
		}
		managedContent = append(managedContent, fmt.Sprintf("    User %s", host.User))
		managedContent = append(managedContent, fmt.Sprintf("    IdentityFile %s", key.Path))
		managedContent = append(managedContent, "    IdentitiesOnly yes")
		if host.Port > 0 {
			managedContent = append(managedContent, fmt.Sprintf("    Port %d", host.Port))
		}
		managedContent = append(managedContent, "")
	}

	managedContent = append(managedContent, skmManagedEnd)
	managedContent = append(managedContent, "")

	// Combine content
	finalContent := existingContent
	if !strings.HasSuffix(finalContent, "\n\n") && finalContent != "" {
		finalContent += "\n"
	}
	finalContent += strings.Join(managedContent, "\n")

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Write config
	if err := os.WriteFile(m.configPath, []byte(finalContent), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}

// RemoveManagedSection removes the SKM managed section from SSH config
func (m *Manager) RemoveManagedSection() error {
	if _, err := os.Stat(m.configPath); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to open SSH config: %w", err)
	}
	defer file.Close()

	var lines []string
	inManagedSection := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, skmManagedStart) {
			inManagedSection = true
			continue
		}

		if strings.Contains(line, skmManagedEnd) {
			inManagedSection = false
			continue
		}

		if !inManagedSection {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read SSH config: %w", err)
	}

	content := strings.Join(lines, "\n")
	if err := os.WriteFile(m.configPath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write SSH config: %w", err)
	}

	return nil
}
