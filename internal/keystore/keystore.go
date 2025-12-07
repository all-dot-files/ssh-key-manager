package keystore

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/all-dot-files/ssh-key-manager/pkg/crypto"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
	"golang.org/x/crypto/ssh"
)

// KeyStore manages SSH keys
type KeyStore struct {
	basePath string
}

// NewKeyStore creates a new KeyStore
func NewKeyStore(basePath string) (*KeyStore, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create keystore directory: %w", err)
	}

	return &KeyStore{
		basePath: basePath,
	}, nil
}

// GenerateKey generates a new SSH key pair
func (ks *KeyStore) GenerateKey(name string, keyType models.KeyType, passphrase string, rsaBits int) (*models.Key, error) {
	var privateKey interface{}
	var err error

	// Generate key based on type
	switch keyType {
	case models.KeyTypeED25519:
		_, privateKey, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ed25519 key: %w", err)
		}

	case models.KeyTypeRSA:
		if rsaBits == 0 {
			rsaBits = 4096
		}
		privateKey, err = rsa.GenerateKey(rand.Reader, rsaBits)
		if err != nil {
			return nil, fmt.Errorf("failed to generate RSA key: %w", err)
		}

	case models.KeyTypeECDSA:
		privateKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	// Marshal private key to PEM
	privateKeyPEM, err := ks.marshalPrivateKey(privateKey, keyType)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Encrypt private key if passphrase is provided
	var privateKeyData []byte
	hasPassphrase := passphrase != ""
	
	if hasPassphrase {
		encrypted, err := crypto.Encrypt(privateKeyPEM, passphrase)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
		// Store as: salt|nonce|ciphertext
		privateKeyData = append(encrypted.Salt, encrypted.Nonce...)
		privateKeyData = append(privateKeyData, encrypted.Ciphertext...)
	} else {
		privateKeyData = privateKeyPEM
	}

	// Generate SSH public key
	sshPrivateKey, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH signer: %w", err)
	}
	publicKey := ssh.MarshalAuthorizedKey(sshPrivateKey.PublicKey())

	// Calculate fingerprint
	fingerprint := ssh.FingerprintSHA256(sshPrivateKey.PublicKey())

	// Write keys to disk
	privatePath := filepath.Join(ks.basePath, name)
	publicPath := privatePath + ".pub"

	if err := os.WriteFile(privatePath, privateKeyData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write private key: %w", err)
	}

	if err := os.WriteFile(publicPath, publicKey, 0644); err != nil {
		return nil, fmt.Errorf("failed to write public key: %w", err)
	}

	// Create Key model
	key := &models.Key{
		Name:          name,
		Type:          keyType,
		Path:          privatePath,
		PubPath:       publicPath,
		Tags:          []string{},
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		Fingerprint:   fingerprint,
		Installed:     false,
		HasPassphrase: hasPassphrase,
	}

	if keyType == models.KeyTypeRSA {
		key.RSABits = rsaBits
	}

	return key, nil
}

// marshalPrivateKey marshals a private key to PEM format
func (ks *KeyStore) marshalPrivateKey(privateKey interface{}, keyType models.KeyType) ([]byte, error) {
	var pemType string
	var keyBytes []byte
	var err error

	switch keyType {
	case models.KeyTypeED25519:
		pemType = "OPENSSH PRIVATE KEY"
		keyBytes, err = x509.MarshalPKCS8PrivateKey(privateKey)

	case models.KeyTypeRSA:
		pemType = "RSA PRIVATE KEY"
		keyBytes = x509.MarshalPKCS1PrivateKey(privateKey.(*rsa.PrivateKey))

	case models.KeyTypeECDSA:
		pemType = "EC PRIVATE KEY"
		keyBytes, err = x509.MarshalECPrivateKey(privateKey.(*ecdsa.PrivateKey))

	default:
		return nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	if err != nil {
		return nil, err
	}

	block := &pem.Block{
		Type:  pemType,
		Bytes: keyBytes,
	}

	return pem.EncodeToMemory(block), nil
}

// LoadPrivateKey loads and optionally decrypts a private key
func (ks *KeyStore) LoadPrivateKey(key *models.Key, passphrase string) ([]byte, error) {
	data, err := os.ReadFile(key.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	if !key.HasPassphrase {
		return data, nil
	}

	// Decrypt
	if len(data) < crypto.SaltLength+crypto.NonceLength {
		return nil, fmt.Errorf("invalid encrypted key data")
	}

	encrypted := &crypto.EncryptedData{
		Salt:       data[:crypto.SaltLength],
		Nonce:      data[crypto.SaltLength : crypto.SaltLength+crypto.NonceLength],
		Ciphertext: data[crypto.SaltLength+crypto.NonceLength:],
	}

	plaintext, err := crypto.Decrypt(encrypted, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt private key: %w", err)
	}

	return plaintext, nil
}

// InstallToSSH installs a key to the SSH directory
func (ks *KeyStore) InstallToSSH(key *models.Key, sshDir string) error {
	// Create target paths
	targetPrivate := filepath.Join(sshDir, "id_"+string(key.Type)+"_"+key.Name)
	targetPublic := targetPrivate + ".pub"

	// Copy private key
	privateData, err := os.ReadFile(key.Path)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	if err := os.WriteFile(targetPrivate, privateData, 0600); err != nil {
		return fmt.Errorf("failed to write private key to SSH dir: %w", err)
	}

	// Copy public key
	publicData, err := os.ReadFile(key.PubPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	if err := os.WriteFile(targetPublic, publicData, 0644); err != nil {
		return fmt.Errorf("failed to write public key to SSH dir: %w", err)
	}

	return nil
}

// ExportPublicKey exports the public key to a file
func (ks *KeyStore) ExportPublicKey(key *models.Key, destination string) error {
	data, err := os.ReadFile(key.PubPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	if err := os.WriteFile(destination, data, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

// DeleteKey deletes a key from the keystore
func (ks *KeyStore) DeleteKey(key *models.Key) error {
	// Delete private key
	if err := os.Remove(key.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	// Delete public key
	if err := os.Remove(key.PubPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete public key: %w", err)
	}

	return nil
}

// GetPublicKeyContent returns the public key content
func (ks *KeyStore) GetPublicKeyContent(key *models.Key) ([]byte, error) {
	return os.ReadFile(key.PubPath)
}
