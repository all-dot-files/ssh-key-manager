package models

import (
	"time"
)

// KeyType represents the type of SSH key
type KeyType string

const (
	KeyTypeED25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
	KeyTypeECDSA   KeyType = "ecdsa"
)

// Key represents an SSH key with its metadata
type Key struct {
	Name      string    `yaml:"name" json:"name"`
	Type      KeyType   `yaml:"type" json:"type"`
	Path      string    `yaml:"path" json:"path"`         // Private key path
	PubPath   string    `yaml:"pub_path" json:"pub_path"` // Public key path
	Tags      []string  `yaml:"tags,omitempty" json:"tags,omitempty"`
	Comment   string    `yaml:"comment,omitempty" json:"comment,omitempty"`
	CreatedAt time.Time `yaml:"created_at" json:"created_at"`
	UpdatedAt time.Time `yaml:"updated_at" json:"updated_at"`
	// Fingerprint of the public key (SHA256)
	Fingerprint string `yaml:"fingerprint,omitempty" json:"fingerprint,omitempty"`
	// Whether this key is installed to ~/.ssh
	Installed bool `yaml:"installed" json:"installed"`
	// RSA key size (only for RSA keys)
	RSABits int `yaml:"rsa_bits,omitempty" json:"rsa_bits,omitempty"`
	// Whether the private key is encrypted with passphrase
	HasPassphrase bool `yaml:"has_passphrase" json:"has_passphrase"`

	// Key rotation tracking
	LastRotatedAt *time.Time `yaml:"last_rotated_at,omitempty" json:"last_rotated_at,omitempty"`
	RotationDueAt *time.Time `yaml:"rotation_due_at,omitempty" json:"rotation_due_at,omitempty"`
	RotatedFrom   string     `yaml:"rotated_from,omitempty" json:"rotated_from,omitempty"` // Previous key name if this is a rotation
}

// KeyRotationStatus represents the rotation status of a key
type KeyRotationStatus string

const (
	RotationStatusOK      KeyRotationStatus = "ok"       // Key is fresh
	RotationStatusWarning KeyRotationStatus = "warning"  // Key is approaching expiration
	RotationStatusExpired KeyRotationStatus = "expired"  // Key should be rotated
)

// GetRotationStatus checks if a key needs rotation based on policy
func (k *Key) GetRotationStatus(policy KeyRotationPolicy) KeyRotationStatus {
	if !policy.Enabled {
		return RotationStatusOK
	}

	baseTime := k.CreatedAt
	if k.LastRotatedAt != nil {
		baseTime = *k.LastRotatedAt
	}

	age := time.Since(baseTime)
	maxAge := time.Duration(policy.MaxKeyAgeMonths) * 30 * 24 * time.Hour
	warnAge := maxAge - time.Duration(policy.WarnBeforeMonths)*30*24*time.Hour

	if age >= maxAge {
		return RotationStatusExpired
	} else if age >= warnAge {
		return RotationStatusWarning
	}

	return RotationStatusOK
}

// GetAgeInMonths returns the age of the key in months
func (k *Key) GetAgeInMonths() int {
	baseTime := k.CreatedAt
	if k.LastRotatedAt != nil {
		baseTime = *k.LastRotatedAt
	}
	return int(time.Since(baseTime).Hours() / 24 / 30)
}

// Host represents an SSH host configuration
type Host struct {
	Host     string   `yaml:"host" json:"host"`
	User     string   `yaml:"user" json:"user"`
	KeyName  string   `yaml:"key" json:"key"` // Reference to Key.Name
	Port     int      `yaml:"port,omitempty" json:"port,omitempty"`
	Hostname string   `yaml:"hostname,omitempty" json:"hostname,omitempty"` // Actual hostname if different
	Tags     []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Device represents a registered device
type Device struct {
	ID           string    `yaml:"id" json:"id"`
	Name         string    `yaml:"name" json:"name"`
	RegisteredAt time.Time `yaml:"registered_at" json:"registered_at"`
	LastSeenAt   time.Time `yaml:"last_seen_at,omitempty" json:"last_seen_at,omitempty"`
	PublicKey    string    `yaml:"public_key,omitempty" json:"public_key,omitempty"` // Device's own public key for key exchange
	Revoked      bool      `yaml:"revoked,omitempty" json:"revoked,omitempty"`
}

// GitRepo represents a Git repository configuration
type GitRepo struct {
	Path    string `yaml:"path" json:"path"`
	Remote  string `yaml:"remote" json:"remote"` // e.g., "origin"
	Host    string `yaml:"host" json:"host"`     // Reference to Host.Host
	User    string `yaml:"user,omitempty" json:"user,omitempty"`
	KeyName string `yaml:"key,omitempty" json:"key,omitempty"` // Override key
}

// KeyPolicy defines how keys should be handled
type KeyPolicy string

const (
	KeyPolicyAuto  KeyPolicy = "auto"  // Automatically use configured key
	KeyPolicyAsk   KeyPolicy = "ask"   // Ask user before using key
	KeyPolicyNever KeyPolicy = "never" // Never auto-use keys
)

// SyncPolicy defines what gets synced to the server
type SyncPolicy struct {
	// SyncPublicKeys enables syncing public keys to server
	SyncPublicKeys bool `yaml:"sync_public_keys" json:"sync_public_keys"`
	// SyncPrivateKeys enables syncing encrypted private keys to server
	SyncPrivateKeys bool `yaml:"sync_private_keys" json:"sync_private_keys"`
	// RequireEncryption requires private keys to be encrypted before upload
	RequireEncryption bool `yaml:"require_encryption" json:"require_encryption"`
}

// KeyRotationPolicy defines key rotation settings
type KeyRotationPolicy struct {
	// Enabled turns on key rotation checks
	Enabled bool `yaml:"enabled" json:"enabled"`
	// MaxKeyAgeMonths is the maximum age of a key in months before rotation warning
	MaxKeyAgeMonths int `yaml:"max_key_age_months" json:"max_key_age_months"`
	// WarnBeforeMonths warns X months before MaxKeyAgeMonths
	WarnBeforeMonths int `yaml:"warn_before_months" json:"warn_before_months"`
	// AutoRotate automatically rotates keys when they expire
	AutoRotate bool `yaml:"auto_rotate" json:"auto_rotate"`
	// NotifyOnRotation sends notification when key needs rotation
	NotifyOnRotation bool `yaml:"notify_on_rotation" json:"notify_on_rotation"`
}

// DefaultKeyRotationPolicy returns default key rotation policy
func DefaultKeyRotationPolicy() KeyRotationPolicy {
	return KeyRotationPolicy{
		Enabled:          true,
		MaxKeyAgeMonths:  24, // 2 years
		WarnBeforeMonths: 3,  // Warn 3 months before
		AutoRotate:       false,
		NotifyOnRotation: true,
	}
}

