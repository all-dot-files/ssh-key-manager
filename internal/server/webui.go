package server

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

//go:embed templates/*.html static/css/* static/js/*
var webContent embed.FS

// WebUI handles the web interface
type WebUI struct {
	server    *Server
	templates *template.Template
}

// NewWebUI creates a new web UI handler
func NewWebUI(server *Server) (*WebUI, error) {
	tmpl, err := template.ParseFS(webContent, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return &WebUI{
		server:    server,
		templates: tmpl,
	}, nil
}

// RegisterRoutes registers web UI routes
func (ui *WebUI) RegisterRoutes(mux *http.ServeMux) {
	// Static files
	mux.Handle("/static/", http.FileServer(http.FS(webContent)))

	// Web pages
	mux.HandleFunc("/", ui.handleIndex)
	mux.HandleFunc("/login", ui.handleLoginPage)
	mux.HandleFunc("/register", ui.handleRegisterPage)
	mux.HandleFunc("/dashboard", ui.requireAuth(ui.handleDashboard))
	mux.HandleFunc("/keys", ui.requireAuth(ui.handleKeys))
	mux.HandleFunc("/devices", ui.requireAuth(ui.handleDevices))
	mux.HandleFunc("/audit", ui.requireAuth(ui.handleAudit))
	mux.HandleFunc("/settings", ui.requireAuth(ui.handleSettings))
}

// handleIndex shows the index page
func (ui *WebUI) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Check if user is logged in
	cookie, err := r.Cookie("auth_token")
	if err == nil && cookie.Value != "" {
		// Verify token
		if ui.verifyToken(cookie.Value) {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleLoginPage shows the login page
func (ui *WebUI) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ui.templates.ExecuteTemplate(w, "login.html", nil)
		return
	}

	// Handle login POST
	username := r.FormValue("username")
	password := r.FormValue("password")

	user, err := ui.server.store.GetUser(username)
	if err != nil || !verifyPassword(user.PasswordHash, password) {
		ui.templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
			"Error": "Invalid username or password",
		})
		return
	}

	// Generate token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, _ := token.SignedString(ui.server.jwtSecret)

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    tokenString,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// handleRegisterPage shows the registration page
func (ui *WebUI) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ui.templates.ExecuteTemplate(w, "register.html", nil)
		return
	}

	// Handle registration POST
	username := r.FormValue("username")
	password := r.FormValue("password")
	email := r.FormValue("email")

	// Validate
	if username == "" || password == "" || email == "" {
		ui.templates.ExecuteTemplate(w, "register.html", map[string]interface{}{
			"Error": "All fields are required",
		})
		return
	}

	// Check if user exists
	_, err := ui.server.store.GetUser(username)
	if err == nil {
		ui.templates.ExecuteTemplate(w, "register.html", map[string]interface{}{
			"Error": "Username already exists",
		})
		return
	}

	// Create user
	user := &User{
		ID:           generateID(),
		Username:     username,
		PasswordHash: hashPassword(password),
		Email:        email,
		CreatedAt:    time.Now(),
	}

	if err := ui.server.store.CreateUser(user); err != nil {
		ui.templates.ExecuteTemplate(w, "register.html", map[string]interface{}{
			"Error": "Failed to create user",
		})
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleDashboard shows the dashboard
func (ui *WebUI) handleDashboard(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := ui.server.store.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	keys, _ := ui.server.store.GetPublicKeys(userID)
	devices, _ := ui.server.store.GetDevices(userID)

	data := map[string]interface{}{
		"User":        user,
		"KeyCount":    len(keys),
		"DeviceCount": len(devices),
	}

	ui.templates.ExecuteTemplate(w, "dashboard.html", data)
}

// handleKeys shows the keys management page
func (ui *WebUI) handleKeys(w http.ResponseWriter, r *http.Request, userID string) {
	keys, err := ui.server.store.GetPublicKeys(userID)
	if err != nil {
		http.Error(w, "Failed to load keys", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Keys": keys,
	}

	ui.templates.ExecuteTemplate(w, "keys.html", data)
}

// handleDevices shows the devices management page
func (ui *WebUI) handleDevices(w http.ResponseWriter, r *http.Request, userID string) {
	devices, err := ui.server.store.GetDevices(userID)
	if err != nil {
		http.Error(w, "Failed to load devices", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Devices": devices,
	}

	ui.templates.ExecuteTemplate(w, "devices.html", data)
}

// handleAudit shows the audit log page
func (ui *WebUI) handleAudit(w http.ResponseWriter, r *http.Request, userID string) {
	// TODO: Implement audit log retrieval
	data := map[string]interface{}{
		"AuditLogs": []map[string]interface{}{},
	}

	ui.templates.ExecuteTemplate(w, "audit.html", data)
}

// handleSettings shows the settings page
func (ui *WebUI) handleSettings(w http.ResponseWriter, r *http.Request, userID string) {
	user, err := ui.server.store.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"User": user,
	}

	ui.templates.ExecuteTemplate(w, "settings.html", data)
}

// requireAuth is middleware that requires authentication
func (ui *WebUI) requireAuth(handler func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		userID := ui.getUserIDFromToken(cookie.Value)
		if userID == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		handler(w, r, userID)
	}
}

// verifyToken verifies a JWT token
func (ui *WebUI) verifyToken(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return ui.server.jwtSecret, nil
	})

	return err == nil && token.Valid
}

// getUserIDFromToken extracts user ID from token
func (ui *WebUI) getUserIDFromToken(tokenString string) string {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return ui.server.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return ""
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ""
	}

	userID, _ := claims["user_id"].(string)
	return userID
}

// Helper functions

func verifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// In case of error, return empty string (this should be handled better in production)
		return ""
	}
	return string(hash)
}

// API endpoints for AJAX requests

// handleAPIKeysDelete handles key deletion via API
func (ui *WebUI) handleAPIKeysDelete(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = strings.TrimPrefix(r.URL.Path, "/api/keys/")

	// TODO: Implement key deletion

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

// handleAPIDevicesRevoke handles device revocation via API
func (ui *WebUI) handleAPIDevicesRevoke(w http.ResponseWriter, r *http.Request, userID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deviceID := strings.TrimPrefix(r.URL.Path, "/api/devices/revoke/")

	if err := ui.server.store.RevokeDevice(userID, deviceID); err != nil {
		http.Error(w, "Failed to revoke device", http.StatusInternalServerError)
		return
	}

	ui.server.store.LogAudit(userID, "device_revoke", fmt.Sprintf("Revoked device: %s", deviceID))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}
