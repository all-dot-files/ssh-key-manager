package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"context"

	"github.com/google/uuid"
	"github.com/all-dot-files/ssh-key-manager/pkg/fileio"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"github.com/all-dot-files/ssh-key-manager/internal/storage"
	"github.com/all-dot-files/ssh-key-manager/internal/storage/sqlite"
	yamlStore "github.com/all-dot-files/ssh-key-manager/internal/storage/yaml"
	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigDir     = ".config/skm"
	DefaultConfigFile    = "config.yaml"
	ProjectConfigFile    = ".skmconfig"
	ProjectConfigFileAlt = ".skmconfig.yaml"
)

// Manager handles configuration persistence
type Manager struct {
	configPath    string
	config        *models.Config
	projectConfig *models.ProjectConfig
	projectPath   string
	fileCache     *fileio.FileCache
	store         storage.Store
}

// NewManager creates a new configuration manager
func NewManager(configPath string) (*Manager, error) {
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, DefaultConfigDir, DefaultConfigFile)
	}

	m := &Manager{
		configPath: configPath,
		fileCache:  fileio.NewFileCache(30 * time.Second),
	}

	return m, nil
}

// GetConfigPath returns the path to the configuration file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// GetConfigDir returns the directory containing the configuration
func (m *Manager) GetConfigDir() string {
	return filepath.Dir(m.configPath)
}

// Load loads the configuration from disk
func (m *Manager) Load() error {
	// First load the raw config file to check settings
	data, err := m.fileCache.Read(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			m.config = models.DefaultConfig()
			// For new config, we default to YAML store
			store, err := yamlStore.NewStore(m.configPath)
			if err != nil {
				return fmt.Errorf("failed to initialize yaml store: %w", err)
			}
			m.store = store
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config models.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	m.config = &config

	// Initialize store based on driver
	if m.config.StorageDriver == "sqlite" {
		dbPath := filepath.Join(filepath.Dir(m.configPath), "skm.db")
		store, err := sqlite.NewStore(dbPath)
		if err != nil {
			return fmt.Errorf("failed to initialize sqlite store: %w", err)
		}
		m.store = store
	} else {
		// Default to YAML store
		store, err := yamlStore.NewStore(m.configPath)
		if err != nil {
			return fmt.Errorf("failed to initialize yaml store: %w", err)
		}
		m.store = store
	}

	return nil
}

// Save saves the configuration to disk
func (m *Manager) Save() error {
	if m.config == nil {
		return fmt.Errorf("no configuration to save")
	}

	m.config.UpdatedAt = time.Now()

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Invalidate cache
	m.fileCache.Invalidate(m.configPath)

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *models.Config {
	if m.config == nil {
		m.config = models.DefaultConfig()
	}
	return m.config
}

// Initialize initializes a new configuration
func (m *Manager) Initialize(deviceName, sshDir string) error {
	// Check if config already exists
	if _, err := os.Stat(m.configPath); err == nil {
		return fmt.Errorf("configuration already exists at %s", m.configPath)
	}

	m.config = models.DefaultConfig()
	m.config.DeviceID = uuid.New().String()
	m.config.DeviceName = deviceName

	// Expand paths
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	m.config.KeystorePath = filepath.Join(home, ".config", "skm", "keys")
	
	if sshDir != "" {
		m.config.SSHDir = sshDir
	} else {
		m.config.SSHDir = filepath.Join(home, ".ssh")
	}

	// Create keystore directory
	if err := os.MkdirAll(m.config.KeystorePath, 0700); err != nil {
		return fmt.Errorf("failed to create keystore directory: %w", err)
	}

	// Save initial configuration
	if err := m.Save(); err != nil {
		return fmt.Errorf("failed to save initial configuration: %w", err)
	}

	// Initialize default store (YAML)
	store, err := yamlStore.NewStore(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize yaml store: %w", err)
	}
	m.store = store

	return nil
}

func (m *Manager) useDB() bool {
	// Logic is now separate, but we keep this helper if needed, 
	// though purely checking m.store != nil is better since we always have a store now.
	return m.config != nil && m.config.StorageDriver == "sqlite" && m.store != nil
}

// AddKey adds a key to the configuration
func (m *Manager) AddKey(key models.Key) error {
	if err := m.checkStore(); err != nil {
		return err
	}
	if err := m.store.Key().Add(context.Background(), key); err != nil {
		return err
	}
	
	return m.reload()
}

// GetKey retrieves a key by name
func (m *Manager) GetKey(name string) (*models.Key, error) {
	if err := m.checkStore(); err != nil {
		return nil, err
	}
	return m.store.Key().Get(context.Background(), name)
}

// RemoveKey removes a key from the configuration
func (m *Manager) RemoveKey(name string) error {
	if err := m.checkStore(); err != nil {
		return err
	}
	if err := m.store.Key().Delete(context.Background(), name); err != nil {
		return err
	}
	
	return m.reload()
}

// AddHost adds a host to the configuration
func (m *Manager) AddHost(host models.Host) error {
	if err := m.checkStore(); err != nil {
		return err
	}
	if err := m.store.Host().Add(context.Background(), host); err != nil {
		return err
	}
	
	return m.reload()
}

// GetHost retrieves a host by hostname
func (m *Manager) GetHost(hostname string) (*models.Host, error) {
	if err := m.checkStore(); err != nil {
		return nil, err
	}
	return m.store.Host().Get(context.Background(), hostname)
}

// AddRepo adds a Git repository configuration
func (m *Manager) AddRepo(repo models.GitRepo) error {
	if err := m.checkStore(); err != nil {
		return err
	}
	if err := m.store.Repo().Add(context.Background(), repo); err != nil {
		return err
	}
	
	return m.reload()
}

// GetRepo retrieves a repository configuration
func (m *Manager) GetRepo(path, remote string) (*models.GitRepo, error) {
	if err := m.checkStore(); err != nil {
		return nil, err
	}
	return m.store.Repo().Get(context.Background(), path)
}

// ListRepos returns all bound repositories
func (m *Manager) ListRepos() ([]models.GitRepo, error) {
	if err := m.checkStore(); err != nil {
		return nil, err
	}
	return m.store.Repo().List(context.Background())
}

// LoadProjectConfig loads project-level configuration from the current or specified directory
func (m *Manager) LoadProjectConfig(dir string) error {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Try to find .skmconfig in current or parent directories
	projectConfigPath := m.findProjectConfig(dir)
	if projectConfigPath == "" {
		// No project config found, that's okay
		return nil
	}

	data, err := m.fileCache.Read(projectConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read project config: %w", err)
	}

	var projectConfig models.ProjectConfig
	if err := yaml.Unmarshal(data, &projectConfig); err != nil {
		return fmt.Errorf("failed to parse project config: %w", err)
	}

	m.projectConfig = &projectConfig
	m.projectPath = filepath.Dir(projectConfigPath)
	return nil
}

// findProjectConfig searches for .skmconfig in current and parent directories
func (m *Manager) findProjectConfig(startDir string) string {
	dir := startDir

	for {
		// Try both .skmconfig and .skmconfig.yaml
		for _, name := range []string{ProjectConfigFile, ProjectConfigFileAlt} {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root
			break
		}
		dir = parent
	}

	return ""
}

// GetMerged returns configuration with project config merged
// Project config takes precedence over global config
func (m *Manager) GetMerged() *models.Config {
	config := m.Get()

	if m.projectConfig == nil {
		return config
	}

	// Create a copy to avoid modifying the original
	merged := *config

	// Override user info if set in project config
	if m.projectConfig.User != "" {
		merged.User = m.projectConfig.User
	}
	if m.projectConfig.Email != "" {
		merged.Email = m.projectConfig.Email
	}

	// Override policy if set
	if m.projectConfig.DefaultKeyPolicy != "" {
		merged.DefaultKeyPolicy = m.projectConfig.DefaultKeyPolicy
	}

	// Override debug mode if set
	if m.projectConfig.Debug != nil {
		merged.Debug = *m.projectConfig.Debug
	}

	// Merge hosts (project hosts take precedence)
	if len(m.projectConfig.Hosts) > 0 {
		hostMap := make(map[string]models.Host)

		// Add global hosts first
		for _, h := range config.Hosts {
			hostMap[h.Host] = h
		}

		// Override with project hosts
		for _, h := range m.projectConfig.Hosts {
			hostMap[h.Host] = h
		}

		// Convert back to slice
		merged.Hosts = make([]models.Host, 0, len(hostMap))
		for _, h := range hostMap {
			merged.Hosts = append(merged.Hosts, h)
		}
	}

	return &merged
}

// GetProjectConfig returns the project configuration if loaded
func (m *Manager) GetProjectConfig() *models.ProjectConfig {
	return m.projectConfig
}

// HasProjectConfig returns true if a project config is loaded
func (m *Manager) HasProjectConfig() bool {
	return m.projectConfig != nil
}

// GetProjectPath returns the path where project config was found
func (m *Manager) GetProjectPath() string {
	return m.projectPath
}

// CreateProjectConfig creates a new project configuration file
func (m *Manager) CreateProjectConfig(dir string, config *models.ProjectConfig) error {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	configPath := filepath.Join(dir, ProjectConfigFile)

	// Check if already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("project configuration already exists: %s", configPath)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal project config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write project config: %w", err)
	}

	return nil
}

// GetEffectiveUser returns the effective user (project or global)
func (m *Manager) GetEffectiveUser() string {
	if m.projectConfig != nil && m.projectConfig.User != "" {
		return m.projectConfig.User
	}
	return m.config.User
}

// GetEffectiveEmail returns the effective email (project or global)
func (m *Manager) GetEffectiveEmail() string {
	if m.projectConfig != nil && m.projectConfig.Email != "" {
		return m.projectConfig.Email
	}
	return m.config.Email
}

// GetEffectiveDebug returns the effective debug mode
func (m *Manager) GetEffectiveDebug() bool {
	if m.projectConfig != nil && m.projectConfig.Debug != nil {
		return *m.projectConfig.Debug
	}
	return m.config.Debug
}

// UpdateKey updates an existing key
func (m *Manager) UpdateKey(name string, key models.Key) error {
	// Update is implicitly supported by Add for most stores or we can assume Add handles upsert
	// But let's check if store supports specific Update or just Add
	// Our store interface has Update for Host but not explicit Update for Key (oops, looking at interface)
	// storage.KeyStore: Add, Get, List, Delete. Add usually implies Upsert in our SQLite impl?
	// Checking SQLite impl: INSERT ... might fail if PK exists.
	// SQLite store.go: INSERT INTO keys ...
	// SQLite impl for Key.Add does INSERT. If PK exists, it fails.
	// So we need to Delete then Add, or fix Store interface to support Update.
	// For now, let's Delete then Add.
	
	// Optimization: If it's a metadata update, just update. 
	if err := m.checkStore(); err != nil {
		return err
	}
	
	// Simple approach: Delete and Add
	if err := m.store.Key().Delete(context.Background(), name); err != nil {
		// Ignore not found if we are just updating? No, strict.
		// However, if we want to support rename, name might change?
		// Assuming name is PK and unchanged here.
		return fmt.Errorf("failed to delete key during update: %w", err)
	}
	
	if err := m.store.Key().Add(context.Background(), key); err != nil {
		return err
	}

	return m.reload()
}



// UpdateHost updates an existing host
func (m *Manager) UpdateHost(name string, host models.Host) error {
	if err := m.checkStore(); err != nil {
		return err
	}
	if err := m.store.Host().Update(context.Background(), host); err != nil {
		return err
	}
	
	return m.reload()
}

// RemoveHost removes a host
func (m *Manager) RemoveHost(name string) error {
	if err := m.checkStore(); err != nil {
		return err
	}
	if err := m.store.Host().Delete(context.Background(), name); err != nil {
		return err
	}
	
	return m.reload()
}



// SetDebug sets the debug mode
func (m *Manager) SetDebug(debug bool) error {
	config := m.Get()
	config.Debug = debug
	return m.Save()
}

// GetDeviceID returns the device ID
func (m *Manager) GetDeviceID() string {
	return m.Get().DeviceID
}

// GetDeviceName returns the device name
func (m *Manager) GetDeviceName() string {
	return m.Get().DeviceName
}

// SetDeviceName sets the device name
func (m *Manager) SetDeviceName(name string) error {
	config := m.Get()
	config.DeviceName = name
	return m.Save()
}

// GetServerURL returns the server URL
func (m *Manager) GetServerURL() string {
	return m.Get().Server
}

// SetServerURL sets the server URL
func (m *Manager) SetServerURL(url string) error {
	config := m.Get()
	config.Server = url
	return m.Save()
}

// GetServerToken returns the server token
func (m *Manager) GetServerToken() string {
	return m.Get().ServerToken
}

// SetServerToken sets the server token
func (m *Manager) SetServerToken(token string) error {
	config := m.Get()
	config.ServerToken = token
	return m.Save()
}

// GetLastSync returns the last sync time
func (m *Manager) GetLastSync() *time.Time {
	return m.Get().LastSync
}

// SetLastSync sets the last sync time
func (m *Manager) SetLastSync(t time.Time) error {
	config := m.Get()
	config.LastSync = &t
	return m.Save()
}


// ListKeys returns all keys
func (m *Manager) ListKeys() ([]models.Key, error) {
	if err := m.checkStore(); err != nil {
		return nil, err
	}
	return m.store.Key().List(context.Background())
}

// ListHosts returns all hosts
func (m *Manager) ListHosts() ([]models.Host, error) {
	if err := m.checkStore(); err != nil {
		return nil, err
	}
	return m.store.Host().List(context.Background())
}

// Helper methods

func (m *Manager) checkStore() error {
	if m.store == nil {
		return fmt.Errorf("store not initialized")
	}
	return nil
}

func (m *Manager) reload() error {
	m.fileCache.Invalidate(m.configPath)
	return m.Load()
}
