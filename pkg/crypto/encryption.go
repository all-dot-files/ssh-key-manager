package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/all-dot-files/ssh-key-manager/pkg/errors"
	"golang.org/x/crypto/argon2"
)

const (
	// Argon2 parameters (OWASP recommended)
	ArgonTime    = 3
	ArgonMemory  = 64 * 1024 // 64 MB
	ArgonThreads = 4
	ArgonKeyLen  = 32

	// Salt length
	SaltLength = 32
	
	// Nonce length for AES-GCM
	NonceLength = 12
)

// EncryptedData represents encrypted data with its salt and nonce
type EncryptedData struct {
	Salt       []byte `json:"salt"`
	Nonce      []byte `json:"nonce"`
	Ciphertext []byte `json:"ciphertext"`
}

// DeriveKey derives a key from a passphrase using Argon2id
func DeriveKey(passphrase string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(passphrase),
		salt,
		ArgonTime,
		ArgonMemory,
		ArgonThreads,
		ArgonKeyLen,
	)
}

// GenerateSalt generates a random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "GenerateSalt", "failed to generate salt")
	}
	return salt, nil
}

// Encrypt encrypts data using AES-256-GCM with a passphrase
func Encrypt(plaintext []byte, passphrase string) (*EncryptedData, error) {
	// Generate salt
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	// Derive key from passphrase
	key := DeriveKey(passphrase, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "Encrypt", "failed to create cipher")
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "Encrypt", "failed to create GCM")
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "Encrypt", "failed to generate nonce")
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return &EncryptedData{
		Salt:       salt,
		Nonce:      nonce,
		Ciphertext: ciphertext,
	}, nil
}

// Decrypt decrypts data using AES-256-GCM with a passphrase
func Decrypt(data *EncryptedData, passphrase string) ([]byte, error) {
	// Derive key from passphrase
	key := DeriveKey(passphrase, data.Salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "Decrypt", "failed to create cipher")
	}

	// Create GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInternal, "Decrypt", "failed to create GCM")
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, data.Nonce, data.Ciphertext, nil)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidInput, "Decrypt", "failed to decrypt")
	}

	return plaintext, nil
}

// EncryptToBase64 encrypts data and returns base64-encoded result
func EncryptToBase64(plaintext []byte, passphrase string) (string, error) {
	encrypted, err := Encrypt(plaintext, passphrase)
	if err != nil {
		return "", err
	}

	// Concatenate salt + nonce + ciphertext
	combined := append(encrypted.Salt, encrypted.Nonce...)
	combined = append(combined, encrypted.Ciphertext...)

	return base64.StdEncoding.EncodeToString(combined), nil
}

// DecryptFromBase64 decrypts base64-encoded encrypted data
func DecryptFromBase64(encoded string, passphrase string) ([]byte, error) {
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrInvalidInput, "DecryptFromBase64", "failed to decode base64")
	}

	if len(combined) < SaltLength+NonceLength {
		return nil, errors.New(errors.ErrInvalidInput, "DecryptFromBase64", "invalid encrypted data: too short")
	}

	data := &EncryptedData{
		Salt:       combined[:SaltLength],
		Nonce:      combined[SaltLength : SaltLength+NonceLength],
		Ciphertext: combined[SaltLength+NonceLength:],
	}

	return Decrypt(data, passphrase)
}

// HashData returns SHA256 hash of data
func HashData(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashDataToString returns SHA256 hash of data as hex string
func HashDataToString(data []byte) string {
	return fmt.Sprintf("%x", HashData(data))
}
