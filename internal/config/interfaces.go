package config

import (
	"time"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// ConfigStore defines the interface for configuration persistence
type ConfigStore interface {
	// Load loads the configuration
	Load() error

	// Save saves the configuration
	Save() error

	// Get returns the current configuration
	Get() *models.Config

	// Initialize initializes a new configuration
	Initialize(deviceName string) error

	// Key operations
	AddKey(key models.Key) error
	GetKey(name string) (*models.Key, error)
	UpdateKey(name string, key models.Key) error
	RemoveKey(name string) error
	ListKeys() []models.Key

	// Host operations
	AddHost(host models.Host) error
	GetHost(name string) (*models.Host, error)
	UpdateHost(name string, host models.Host) error
	RemoveHost(name string) error
	ListHosts() []models.Host

	// Project-specific config
	LoadProjectConfig(dir string) error
	HasProjectConfig() bool
	GetProjectPath() string

	// Settings
	GetEffectiveDebug() bool
	SetDebug(debug bool) error

	// Device management
	GetDeviceID() string
	GetDeviceName() string
	SetDeviceName(name string) error

	// Server settings
	GetServerURL() string
	SetServerURL(url string) error
	GetServerToken() string
	SetServerToken(token string) error

	// Audit
	GetLastSync() *time.Time
	SetLastSync(t time.Time) error
}

