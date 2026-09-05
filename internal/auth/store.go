package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/buildshare/cli/internal/config"
)

// Credentials stores the user's auth state on disk.
type Credentials struct {
	Token  string `json:"token"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	UserID int    `json:"userId"`
}

// Save writes credentials to the config directory with restrictive permissions.
func Save(creds *Credentials) error {
	dir := config.UserConfigDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return err
	}

	path := config.CredentialsFilePath()
	return os.WriteFile(path, data, 0600)
}

// Load reads stored credentials. Returns nil if none exist.
func Load() *Credentials {
	path := config.CredentialsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var creds Credentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil
	}
	return &creds
}

// Clear removes stored credentials.
func Clear() error {
	path := config.CredentialsFilePath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.Remove(path)
}

// ResolveToken returns the active token, checking env var first, then stored credentials.
func ResolveToken() string {
	if t := os.Getenv("BUILDSHARE_TOKEN"); t != "" {
		return t
	}
	creds := Load()
	if creds != nil {
		return creds.Token
	}
	return ""
}

// IsLoggedIn returns true if a valid token is available.
func IsLoggedIn() bool {
	return ResolveToken() != ""
}

// ConfigDir ensures the config directory exists and returns its path.
func ConfigDir() string {
	dir := config.UserConfigDir()
	_ = os.MkdirAll(dir, 0700)
	return dir
}

// InitProjectConfig creates a .buildshare.yaml in the current directory.
func InitProjectConfig(appID, appName string) error {
	content := fmt.Sprintf("# BuildShare project configuration\nproject: %s\napp_id: %s\n", appName, appID)
	return os.WriteFile(filepath.Join(".", ".buildshare.yaml"), []byte(content), 0644)
}
