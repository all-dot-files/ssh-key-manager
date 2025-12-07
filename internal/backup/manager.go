package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DefaultBackupDir is the directory where backups are stored relative to config dir
const DefaultBackupDir = "backups"

// Manager handles backup and restore operations
type Manager struct {
	configDir string // The directory to back up (e.g., ~/.config/skm)
	backupDir string // The directory where backups are stored
}

// NewManager creates a new backup manager
func NewManager(configDir string) (*Manager, error) {
	backupDir := filepath.Join(configDir, DefaultBackupDir)
	return &Manager{
		configDir: configDir,
		backupDir: backupDir,
	}, nil
}

// EnsureBackupDir ensures the backup directory exists
func (m *Manager) EnsureBackupDir() error {
	if err := os.MkdirAll(m.backupDir, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}
	return nil
}

// Create creates a new backup of the configuration directory
func (m *Manager) Create(message string) (string, error) {
	if err := m.EnsureBackupDir(); err != nil {
		return "", err
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("skm-backup-%s.zip", timestamp)
	if message != "" {
		// sanitize message for filename
		safeMsg := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				return r
			}
			return '-'
		}, message)
		filename = fmt.Sprintf("skm-backup-%s-%s.zip", timestamp, safeMsg)
	}

	destPath := filepath.Join(m.backupDir, filename)

	// Create zip file
	zipFile, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer zipFile.Close()

	archive := zip.NewWriter(zipFile)
	defer archive.Close()

	// Walk through the config directory
	err = filepath.Walk(m.configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip backup directory itself to avoid recursion
		if path == m.backupDir || strings.HasPrefix(path, m.backupDir) {
			return nil
		}

		// Get relative path for the header
		relPath, err := filepath.Rel(m.configDir, path)
		if err != nil {
			return err
		}

		// Don't include the root folder itself
		if relPath == "." {
			return nil
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set name to relative path
		header.Name = relPath

		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return "", fmt.Errorf("failed to walk config directory: %w", err)
	}

	return destPath, nil
}

// BackupInfo contains info about a backup
type BackupInfo struct {
	Name      string
	Path      string
	Size      int64
	Timestamp time.Time
}

// List returns a list of available backups
func (m *Manager) List() ([]BackupInfo, error) {
	if _, err := os.Stat(m.backupDir); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	entries, err := os.ReadDir(m.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var backups []BackupInfo
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".zip") {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			backups = append(backups, BackupInfo{
				Name:      entry.Name(),
				Path:      filepath.Join(m.backupDir, entry.Name()),
				Size:      info.Size(),
				Timestamp: info.ModTime(),
			})
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

// Restore restores the configuration from a backup file
func (m *Manager) Restore(backupPath string) error {
	// Open zip reader
	reader, err := zip.OpenReader(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer reader.Close()

	// Extract files
	for _, file := range reader.File {
		path := filepath.Join(m.configDir, file.Name)

		// Check for Zip Slip
		if !strings.HasPrefix(path, filepath.Clean(m.configDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("failed to open file for writing: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("failed to write file content: %w", err)
		}
	}

	return nil
}

// Delete deletes a backup file
func (m *Manager) Delete(name string) error {
	path := filepath.Join(m.backupDir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("backup %s not found", name)
	}
	return os.Remove(path)
}
