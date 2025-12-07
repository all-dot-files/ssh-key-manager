package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/all-dot-files/ssh-key-manager/internal/api"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// Server represents the SKM HTTP server
type Server struct {
	addr      string
	jwtSecret []byte
	store     Store
	webUI     *WebUI
}

// Store defines the interface for data persistence
type Store interface {
	// User operations
	GetUser(username string) (*User, error)
	GetUserByID(userID string) (*User, error)
	CreateUser(user *User) error

	// Device operations
	RegisterDevice(userID string, device *models.Device) error
	GetDevices(userID string) ([]models.Device, error)
	RevokeDevice(userID, deviceID string) error

	// Key operations
	SavePublicKeys(userID string, keys []api.PublicKeyData) error
	GetPublicKeys(userID string) ([]api.PublicKeyData, error)
	SavePrivateKeys(userID string, keys []api.PrivateKeyData) error
	GetPrivateKeys(userID string) ([]api.PrivateKeyData, error)

	// Audit
	LogAudit(userID, action, details string) error
	GetAuditLogs(userID string, limit int) ([]interface{}, error)
}

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"` // Store in file, but exclude from API responses
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserResponse represents a user for API responses (without password)
type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// NewServer creates a new SKM server instance
func NewServer(addr string, jwtSecret []byte, store Store) (*Server, error) {
	server := &Server{
		addr:      addr,
		jwtSecret: jwtSecret,
		store:     store,
	}

	// Initialize Web UI
	webUI, err := NewWebUI(server)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize web UI: %w", err)
	}
	server.webUI = webUI

	return server, nil
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Register Web UI routes
	s.webUI.RegisterRoutes(mux)

	// Auth endpoints
	mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("/api/v1/auth/register", s.handleRegister)

	// Device endpoints
	mux.HandleFunc("/api/v1/devices/register", s.withAuth(s.handleDeviceRegister))
	mux.HandleFunc("/api/v1/devices", s.withAuth(s.handleGetDevices))
	mux.HandleFunc("/api/v1/devices/", s.withAuth(s.handleDeviceOperation))

	// Key endpoints
	mux.HandleFunc("/api/v1/keys/public", s.withAuth(s.handlePublicKeys))
	mux.HandleFunc("/api/v1/keys/private", s.withAuth(s.handlePrivateKeys))

	// Audit endpoints
	mux.HandleFunc("/api/v1/audit", s.withAuth(s.handleGetAudit))

	fmt.Printf("SKM Server starting on %s\n", s.addr)
	fmt.Printf("Web UI available at: http://%s/\n", s.addr)
	fmt.Printf("API available at: http://%s/api/v1/\n", s.addr)
	return http.ListenAndServe(s.addr, mux)
}

// handleLogin handles user login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// TODO: Verify password (this is a simplified example)
	user, err := s.store.GetUser(req.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	s.store.LogAudit(user.ID, "login", "User logged in")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

// handleRegister handles user registration
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// TODO: Hash password properly
	user := &User{
		ID:           generateID(),
		Username:     req.Username,
		PasswordHash: req.Password, // Should be hashed!
		Email:        req.Email,
		CreatedAt:    time.Now(),
	}

	if err := s.store.CreateUser(user); err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"id":      user.ID,
		"message": "User created successfully",
	})
}

// handleDeviceRegister registers a device
func (s *Server) handleDeviceRegister(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var device models.Device
	if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	device.RegisteredAt = time.Now()
	device.LastSeenAt = time.Now()

	if err := s.store.RegisterDevice(userID, &device); err != nil {
		http.Error(w, "Failed to register device", http.StatusInternalServerError)
		return
	}

	s.store.LogAudit(userID, "device_register", fmt.Sprintf("Device registered: %s", device.Name))

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(device)
}

// handleGetDevices retrieves all devices
func (s *Server) handleGetDevices(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	devices, err := s.store.GetDevices(userID)
	if err != nil {
		http.Error(w, "Failed to retrieve devices", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

// handleDeviceOperation handles device-specific operations
func (s *Server) handleDeviceOperation(w http.ResponseWriter, r *http.Request, userID string) {
	// Extract device ID from path
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	deviceID := parts[4]
	operation := ""
	if len(parts) > 5 {
		operation = parts[5]
	}

	if operation == "revoke" && r.Method == http.MethodPost {
		if err := s.store.RevokeDevice(userID, deviceID); err != nil {
			http.Error(w, "Failed to revoke device", http.StatusInternalServerError)
			return
		}

		s.store.LogAudit(userID, "device_revoke", fmt.Sprintf("Device revoked: %s", deviceID))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Device revoked"})
		return
	}

	http.Error(w, "Invalid operation", http.StatusBadRequest)
}

// handlePublicKeys handles public key operations
func (s *Server) handlePublicKeys(w http.ResponseWriter, r *http.Request, userID string) {
	switch r.Method {
	case http.MethodGet:
		keys, err := s.store.GetPublicKeys(userID)
		if err != nil {
			http.Error(w, "Failed to retrieve keys", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)

	case http.MethodPost:
		var keys []api.PublicKeyData
		if err := json.NewDecoder(r.Body).Decode(&keys); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := s.store.SavePublicKeys(userID, keys); err != nil {
			http.Error(w, "Failed to save keys", http.StatusInternalServerError)
			return
		}

		s.store.LogAudit(userID, "keys_push", fmt.Sprintf("Pushed %d public keys", len(keys)))

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Keys saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePrivateKeys handles private key operations
func (s *Server) handlePrivateKeys(w http.ResponseWriter, r *http.Request, userID string) {
	switch r.Method {
	case http.MethodGet:
		keys, err := s.store.GetPrivateKeys(userID)
		if err != nil {
			http.Error(w, "Failed to retrieve keys", http.StatusInternalServerError)
			return
		}

		s.store.LogAudit(userID, "keys_pull_private", fmt.Sprintf("Pulled %d private keys", len(keys)))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)

	case http.MethodPost:
		var keys []api.PrivateKeyData
		if err := json.NewDecoder(r.Body).Decode(&keys); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if err := s.store.SavePrivateKeys(userID, keys); err != nil {
			http.Error(w, "Failed to save keys", http.StatusInternalServerError)
			return
		}

		s.store.LogAudit(userID, "keys_push_private", fmt.Sprintf("Pushed %d private keys", len(keys)))

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "Keys saved"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// withAuth wraps a handler with JWT authentication
func (s *Server) withAuth(handler func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization required", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return s.jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		handler(w, r, userID)
	}
}

// generateID generates a unique ID (simplified)
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// handleGetAudit retrieves audit logs
func (s *Server) handleGetAudit(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get limit from query params
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	logs, err := s.store.GetAuditLogs(userID, limit)
	if err != nil {
		http.Error(w, "Failed to retrieve audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
