package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/all-dot-files/ssh-key-manager/internal/api"
	"github.com/all-dot-files/ssh-key-manager/internal/models"
)

// GinServer wraps the gin engine with server dependencies
type GinServer struct {
	engine    *gin.Engine
	jwtSecret []byte
	store     Store
}

// TokenAuthMiddleware 校验 header 里的 token 或 cookie 里的 token
func (gs *GinServer) TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 首先尝试从 header 获取 token
		tokenString := c.GetHeader("Authorization")
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		}

		// 如果 header 中没有，尝试从 cookie 获取
		if tokenString == "" {
			cookie, err := c.Cookie("auth_token")
			if err == nil {
				tokenString = cookie
			}
		}

		if tokenString == "" {
			// 判断是 API 请求还是页面请求
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			} else {
				c.Redirect(http.StatusSeeOther, "/login")
			}
			c.Abort()
			return
		}
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return gs.jwtSecret, nil
		})
		if err != nil || !token.Valid {
			// 判断是 API 请求还是页面请求
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			} else {
				c.Redirect(http.StatusSeeOther, "/login")
			}
			c.Abort()
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.Abort()
			return
		}
		c.Set("claims", claims)
		c.Set("user_id", claims["user_id"].(string))
		c.Set("username", claims["username"].(string))
		c.Next()
	}
}

// NewGinServer creates a new gin-based server
func NewGinServer(jwtSecret []byte, store Store) *GinServer {
	gs := &GinServer{
		engine:    gin.Default(),
		jwtSecret: jwtSecret,
		store:     store,
	}

	gs.setupRoutes()
	return gs
}

// setupRoutes configures all routes
func (gs *GinServer) setupRoutes() {
	r := gs.engine
	
	r.SetFuncMap(template.FuncMap{
		"upper": strings.ToUpper,
	})

	r.LoadHTMLGlob("internal/server/templates/*.html")
	r.Static("/static", "internal/server/static")

	// Public routes
	r.GET("/", gs.handleIndex)
	r.GET("/login", gs.handleLoginPage)
	r.POST("/login", gs.handleLogin)
	r.GET("/register", gs.handleRegisterPage)
	r.POST("/register", gs.handleRegister)
	r.GET("/logout", gs.handleLogout)

	// Protected web pages
	r.GET("/dashboard", gs.TokenAuthMiddleware(), gs.handleDashboard)
	r.GET("/keys", gs.TokenAuthMiddleware(), gs.handleKeys)
	r.GET("/devices", gs.TokenAuthMiddleware(), gs.handleDevices)
	r.GET("/audit", gs.TokenAuthMiddleware(), gs.handleAudit)
	r.GET("/settings", gs.TokenAuthMiddleware(), gs.handleSettings)

	// API routes
	api := r.Group("/api/v1")
	{
		// Auth
		api.POST("/auth/login", gs.handleAPILogin)
		api.POST("/auth/register", gs.handleAPIRegister)

		// Protected API routes
		protected := api.Group("", gs.TokenAuthMiddleware())
		{
			// Devices
			protected.POST("/devices/register", gs.handleDeviceRegister)
			protected.GET("/devices", gs.handleGetDevices)
			protected.POST("/devices/:id/revoke", gs.handleRevokeDevice)

			// Keys
			protected.POST("/keys/public", gs.handleSavePublicKeys)
			protected.GET("/keys/public", gs.handleGetPublicKeys)
			protected.POST("/keys/private", gs.handleSavePrivateKeys)
			protected.GET("/keys/private", gs.handleGetPrivateKeys)

			// Audit
			protected.GET("/audit", gs.handleGetAuditLogs)
		}
	}
}

// Run starts the gin server
func (gs *GinServer) Run(addr string) error {
	fmt.Printf("SKM Server starting on %s\n", addr)
	fmt.Printf("Web UI available at: http://%s/\n", addr)
	fmt.Printf("API available at: http://%s/api/v1/\n", addr)
	return gs.engine.Run(addr)
}

// Web handlers

func (gs *GinServer) handleIndex(c *gin.Context) {
	// Check if user has valid token in header
	tokenString := c.GetHeader("Authorization")
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}
	if tokenString != "" {
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return gs.jwtSecret, nil
		})
		if err == nil && token.Valid {
			c.Redirect(http.StatusSeeOther, "/dashboard")
			return
		}
	}
	// Show a simple landing page that redirects to login
	c.HTML(http.StatusOK, "index.html", nil)
}

func (gs *GinServer) handleLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func (gs *GinServer) handleLogin(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	user, err := gs.store.GetUser(username)
	if err != nil || !verifyPassword(user.PasswordHash, password) {
		c.HTML(http.StatusOK, "login.html", gin.H{
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

	tokenString, _ := token.SignedString(gs.jwtSecret)

	gs.store.LogAudit(user.ID, "login", "User logged in via web")

	// Set cookie for web UI
	c.SetCookie(
		"auth_token", // name
		tokenString,  // value
		24*60*60,     // maxAge (24 hours)
		"/",          // path
		"",           // domain
		false,        // secure (set to true in production with HTTPS)
		false,        // httpOnly (false to allow JavaScript access)
	)

	c.Redirect(http.StatusSeeOther, "/dashboard")
}

func (gs *GinServer) handleRegisterPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func (gs *GinServer) handleLogout(c *gin.Context) {
	// Clear the auth cookie
	c.SetCookie(
		"auth_token", // name
		"",           // value
		-1,           // maxAge (delete cookie)
		"/",          // path
		"",           // domain
		false,        // secure
		true,         // httpOnly
	)
	c.Redirect(http.StatusSeeOther, "/login")
}

func (gs *GinServer) handleRegister(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	email := c.PostForm("email")

	if username == "" || password == "" || email == "" {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"Error": "All fields are required",
		})
		return
	}

	// Check if user exists
	_, err := gs.store.GetUser(username)
	if err == nil {
		c.HTML(http.StatusOK, "register.html", gin.H{
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

	if err := gs.store.CreateUser(user); err != nil {
		c.HTML(http.StatusOK, "register.html", gin.H{
			"Error": "Failed to create user",
		})
		return
	}

	c.Redirect(http.StatusSeeOther, "/login")
}

func (gs *GinServer) handleDashboard(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := gs.store.GetUserByID(userID)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	keys, _ := gs.store.GetPublicKeys(userID)
	devices, _ := gs.store.GetDevices(userID)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"User":        user,
		"KeyCount":    len(keys),
		"DeviceCount": len(devices),
		"ActivePage":  "dashboard",
	})
}

func (gs *GinServer) handleKeys(c *gin.Context) {
	userID := c.GetString("user_id")

	keys, err := gs.store.GetPublicKeys(userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load keys")
		return
	}

	user, _ := gs.store.GetUserByID(userID)

	c.HTML(http.StatusOK, "keys.html", gin.H{
		"Keys":       keys,
		"User":       user,
		"ActivePage": "keys",
	})
}

func (gs *GinServer) handleDevices(c *gin.Context) {
	userID := c.GetString("user_id")

	devices, err := gs.store.GetDevices(userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to load devices")
		return
	}

	user, _ := gs.store.GetUserByID(userID)

	c.HTML(http.StatusOK, "devices.html", gin.H{
		"Devices":    devices,
		"User":       user,
		"ActivePage": "devices",
	})
}

func (gs *GinServer) handleAudit(c *gin.Context) {
	userID := c.GetString("user_id")

	logs, err := gs.store.GetAuditLogs(userID, 100)
	if err != nil {
		logs = []interface{}{}
	}

	user, _ := gs.store.GetUserByID(userID)

	c.HTML(http.StatusOK, "audit.html", gin.H{
		"AuditLogs":  logs,
		"User":       user,
		"ActivePage": "audit",
	})
}

func (gs *GinServer) handleSettings(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := gs.store.GetUserByID(userID)
	if err != nil {
		c.String(http.StatusNotFound, "User not found")
		return
	}

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"User":       user,
		"ActivePage": "settings",
	})
}

// API handlers

func (gs *GinServer) handleAPILogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, err := gs.store.GetUser(req.Username)
	if err != nil || !verifyPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(gs.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	gs.store.LogAudit(user.ID, "login", "User logged in via API")

	// Set cookie for web UI
	c.SetCookie(
		"auth_token", // name
		tokenString,  // value
		24*60*60,     // maxAge (24 hours)
		"/",          // path
		"",           // domain
		false,        // secure (set to true in production with HTTPS)
		false,        // httpOnly (false to allow JavaScript access)
	)

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func (gs *GinServer) handleAPIRegister(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user := &User{
		ID:           generateID(),
		Username:     req.Username,
		PasswordHash: hashPassword(req.Password),
		Email:        req.Email,
		CreatedAt:    time.Now(),
	}

	if err := gs.store.CreateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":      user.ID,
		"message": "User created successfully",
	})
}

func (gs *GinServer) handleDeviceRegister(c *gin.Context) {
	userID := c.GetString("user_id")

	var device models.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	device.RegisteredAt = time.Now()
	device.LastSeenAt = time.Now()

	if err := gs.store.RegisterDevice(userID, &device); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device"})
		return
	}

	gs.store.LogAudit(userID, "device_register", fmt.Sprintf("Device registered: %s", device.Name))

	c.JSON(http.StatusCreated, device)
}

func (gs *GinServer) handleGetDevices(c *gin.Context) {
	userID := c.GetString("user_id")

	devices, err := gs.store.GetDevices(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve devices"})
		return
	}

	c.JSON(http.StatusOK, devices)
}

func (gs *GinServer) handleRevokeDevice(c *gin.Context) {
	userID := c.GetString("user_id")
	deviceID := c.Param("id")

	if err := gs.store.RevokeDevice(userID, deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke device"})
		return
	}

	gs.store.LogAudit(userID, "device_revoke", fmt.Sprintf("Revoked device: %s", deviceID))

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (gs *GinServer) handleSavePublicKeys(c *gin.Context) {
	userID := c.GetString("user_id")

	var keys []api.PublicKeyData
	if err := c.ShouldBindJSON(&keys); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := gs.store.SavePublicKeys(userID, keys); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save keys"})
		return
	}

	gs.store.LogAudit(userID, "keys_save", fmt.Sprintf("Saved %d public keys", len(keys)))

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (gs *GinServer) handleGetPublicKeys(c *gin.Context) {
	userID := c.GetString("user_id")

	keys, err := gs.store.GetPublicKeys(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve keys"})
		return
	}

	c.JSON(http.StatusOK, keys)
}

func (gs *GinServer) handleSavePrivateKeys(c *gin.Context) {
	userID := c.GetString("user_id")

	var keys []api.PrivateKeyData
	if err := c.ShouldBindJSON(&keys); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := gs.store.SavePrivateKeys(userID, keys); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save keys"})
		return
	}

	gs.store.LogAudit(userID, "keys_save", fmt.Sprintf("Saved %d private keys", len(keys)))

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

func (gs *GinServer) handleGetPrivateKeys(c *gin.Context) {
	userID := c.GetString("user_id")

	keys, err := gs.store.GetPrivateKeys(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve keys"})
		return
	}

	c.JSON(http.StatusOK, keys)
}

func (gs *GinServer) handleGetAuditLogs(c *gin.Context) {
	userID := c.GetString("user_id")

	logs, err := gs.store.GetAuditLogs(userID, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	c.JSON(http.StatusOK, logs)
}
