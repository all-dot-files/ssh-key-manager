package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// Client is an API client for the SKM server
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient creates a new API client
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		token: token,
	}
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) {
	c.token = token
}

// RegisterDevice registers a new device
func (c *Client) RegisterDevice(device *models.Device) error {
	return c.doRequest("POST", "/api/v1/devices/register", device, nil)
}

// SyncPublicKeys uploads public keys to the server
func (c *Client) SyncPublicKeys(keys []PublicKeyData) error {
	return c.doRequest("POST", "/api/v1/keys/public", keys, nil)
}

// SyncPrivateKeys uploads encrypted private keys to the server
func (c *Client) SyncPrivateKeys(keys []PrivateKeyData) error {
	return c.doRequest("POST", "/api/v1/keys/private", keys, nil)
}

// FetchPublicKeys retrieves public keys from the server
func (c *Client) FetchPublicKeys() ([]PublicKeyData, error) {
	var keys []PublicKeyData
	if err := c.doRequest("GET", "/api/v1/keys/public", nil, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

// FetchPrivateKeys retrieves encrypted private keys from the server
func (c *Client) FetchPrivateKeys() ([]PrivateKeyData, error) {
	var keys []PrivateKeyData
	if err := c.doRequest("GET", "/api/v1/keys/private", nil, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

// GetDevices retrieves all devices for the current user
func (c *Client) GetDevices() ([]models.Device, error) {
	var devices []models.Device
	if err := c.doRequest("GET", "/api/v1/devices", nil, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// RevokeDevice revokes a device
func (c *Client) RevokeDevice(deviceID string) error {
	return c.doRequest("POST", fmt.Sprintf("/api/v1/devices/%s/revoke", deviceID), nil, nil)
}

// Login authenticates and retrieves a token
func (c *Client) Login(username, password string) (string, error) {
	req := map[string]string{
		"username": username,
		"password": password,
	}
	
	var resp struct {
		Token string `json:"token"`
	}
	
	if err := c.doRequest("POST", "/api/v1/auth/login", req, &resp); err != nil {
		return "", err
	}
	
	c.token = resp.Token
	return resp.Token, nil
}

// doRequest performs an HTTP request
func (c *Client) doRequest(method, path string, body, result interface{}) error {
	var bodyReader io.Reader
	
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// PublicKeyData represents public key data for sync
type PublicKeyData struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	Tags        []string  `json:"tags,omitempty"`
	Comment     string    `json:"comment,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// PrivateKeyData represents encrypted private key data for sync
type PrivateKeyData struct {
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	EncryptedPrivate  string    `json:"encrypted_private"` // Base64 encoded encrypted data
	PublicKey         string    `json:"public_key"`
	Fingerprint       string    `json:"fingerprint"`
	EncryptionMethod  string    `json:"encryption_method"` // e.g., "age", "aes-gcm"
	RecipientDeviceID string    `json:"recipient_device_id,omitempty"` // For device-specific encryption
	CreatedAt         time.Time `json:"created_at"`
}
