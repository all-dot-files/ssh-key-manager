package models

import (
	"time"
)

// Config represents the main SKM configuration
type Config struct {
	// Device configuration
	DeviceID   string `yaml:"device_id" json:"device_id"`
	DeviceName string `yaml:"device_name,omitempty" json:"device_name,omitempty"`

	// User information
	User  string `yaml:"user,omitempty" json:"user,omitempty"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`

	// Server configuration
	Server      string `yaml:"server,omitempty" json:"server,omitempty"`
	ServerToken string `yaml:"server_token,omitempty" json:"server_token,omitempty"` // JWT token

	// Storage configuration
	StorageDriver string `yaml:"storage_driver,omitempty" json:"storage_driver,omitempty"` // "yaml" or "sqlite"

	// Local paths
	KeystorePath string `yaml:"keystore_path" json:"keystore_path"`
	SSHDir       string `yaml:"ssh_dir" json:"ssh_dir"`

	// Policies
	DefaultKeyPolicy KeyPolicy  `yaml:"default_key_policy" json:"default_key_policy"`
	SyncPolicy       SyncPolicy `yaml:"sync_policy" json:"sync_policy"`

	// Key rotation policy
	KeyRotationPolicy KeyRotationPolicy `yaml:"key_rotation_policy,omitempty" json:"key_rotation_policy,omitempty"`

	// Debug mode
	Debug bool `yaml:"debug,omitempty" json:"debug,omitempty"`

	// Data
	Keys     []Key     `yaml:"keys,omitempty" json:"keys,omitempty"`
	Hosts    []Host    `yaml:"hosts,omitempty" json:"hosts,omitempty"`
	Repos    []GitRepo `yaml:"repos,omitempty" json:"repos,omitempty"`
	Devices  []Device  `yaml:"devices,omitempty" json:"devices,omitempty"`
	Projects []string  `yaml:"projects,omitempty" json:"projects,omitempty"` // Project paths

	// Sync metadata
	LastSync *time.Time `yaml:"last_sync,omitempty" json:"last_sync,omitempty"`

	// Metadata
	Version   string    `yaml:"version" json:"version"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
}

// ProjectConfig represents a project-level SKM configuration (.skmconfig)
// This is a subset of Config that can be defined at the project level
type ProjectConfig struct {
	// Project identification
	ProjectName string `yaml:"project_name,omitempty" json:"project_name,omitempty"`

	// Override user information for this project
	User  string `yaml:"user,omitempty" json:"user,omitempty"`
	Email string `yaml:"email,omitempty" json:"email,omitempty"`

	// Override default key for this project
	DefaultKey string `yaml:"default_key,omitempty" json:"default_key,omitempty"`

	// Project-specific hosts
	Hosts []Host `yaml:"hosts,omitempty" json:"hosts,omitempty"`

	// Project-specific Git configuration
	GitUser  string `yaml:"git_user,omitempty" json:"git_user,omitempty"`
	GitEmail string `yaml:"git_email,omitempty" json:"git_email,omitempty"`

	// Policies can be overridden
	DefaultKeyPolicy KeyPolicy `yaml:"default_key_policy,omitempty" json:"default_key_policy,omitempty"`

	// Auto-create settings
	AutoCreateHost bool    `yaml:"auto_create_host,omitempty" json:"auto_create_host,omitempty"`
	AutoCreateKey  bool    `yaml:"auto_create_key,omitempty" json:"auto_create_key,omitempty"`
	KeyType        KeyType `yaml:"key_type,omitempty" json:"key_type,omitempty"`

	// Team sharing info
	TeamName string `yaml:"team_name,omitempty" json:"team_name,omitempty"`

	// Debug mode override
	Debug *bool `yaml:"debug,omitempty" json:"debug,omitempty"`
}

// DefaultConfig returns a new Config with default values
func DefaultConfig() *Config {
	return &Config{
		KeystorePath:     "~/.config/skm/keys",
		SSHDir:           "~/.ssh",
		DefaultKeyPolicy: KeyPolicyAsk,
		SyncPolicy: SyncPolicy{
			SyncPublicKeys:    true,
			SyncPrivateKeys:   false,
			RequireEncryption: true,
		},
		KeyRotationPolicy: DefaultKeyRotationPolicy(),
		Debug:             false,
		Version:           "1.0.0",
		Keys:              []Key{},
		Hosts:             []Host{},
		Repos:             []GitRepo{},
		Devices:           []Device{},
		UpdatedAt:         time.Now(),
	}
}
