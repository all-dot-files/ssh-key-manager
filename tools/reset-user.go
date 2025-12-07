package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user (matching server.User structure)
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func createOrUpdateUser(basePath, username, password, email string) error {
	// Hash the password
	hash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if user file exists
	userFile := filepath.Join(basePath, "users", username+".json")
	var user User

	// Try to read existing user
	data, err := os.ReadFile(userFile)
	if err == nil {
		// User exists, update password
		if err := json.Unmarshal(data, &user); err != nil {
			return fmt.Errorf("failed to parse existing user: %w", err)
		}
		user.PasswordHash = hash
		if email != "" {
			user.Email = email
		}
		fmt.Printf("Updating existing user: %s\n", username)
	} else {
		// Create new user
		user = User{
			ID:           fmt.Sprintf("%d", time.Now().UnixNano()),
			Username:     username,
			PasswordHash: hash,
			Email:        email,
			CreatedAt:    time.Now(),
		}
		fmt.Printf("Creating new user: %s\n", username)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Join(basePath, "users"), 0700); err != nil {
		return fmt.Errorf("failed to create users directory: %w", err)
	}

	// Save user
	data, err = json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	if err := os.WriteFile(userFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write user file: %w", err)
	}

	fmt.Printf("User %s saved successfully!\n", username)
	fmt.Printf("Password: %s\n", password)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run reset-user.go <data-path>")
		fmt.Println("")
		fmt.Println("This will create/update default users (admin/admin, test/test)")
		os.Exit(1)
	}

	basePath := os.Args[1]

	// Create default users
	users := []struct {
		username string
		password string
		email    string
	}{
		{"admin", "admin", "admin@example.com"},
		{"test", "test", "test@example.com"},
	}

	fmt.Printf("Updating users in: %s\n\n", basePath)

	for _, u := range users {
		if err := createOrUpdateUser(basePath, u.username, u.password, u.email); err != nil {
			fmt.Printf("Error updating user %s: %v\n", u.username, err)
		}
		fmt.Println()
	}

	fmt.Println("All users updated successfully!")
	fmt.Println("\nDefault credentials:")
	fmt.Println("  admin / admin")
	fmt.Println("  test / test")
}
