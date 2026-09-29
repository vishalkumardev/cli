package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

// Config holds all resolved configuration values.
type Config struct {
	Token   string
	Project string
	Debug   bool
	JSON    bool
	CI      bool
}

const APIURL = "https://api.buildshare.in/api/v1"
const DefaultAPIURL = APIURL

// Load resolves configuration with this precedence:
// CLI flags (already bound via viper) → env vars → project file → user config → defaults
func Load() *Config {
	viper.SetEnvPrefix("BUILDSHARE")
	viper.AutomaticEnv()

	// Load project-level .buildshare.yaml
	viper.SetConfigName(".buildshare")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	_ = viper.ReadInConfig() // silently ignore missing file

	// Load user-level config as fallback
	viper.SetConfigName("config")
	viper.AddConfigPath(UserConfigDir())
	_ = viper.MergeInConfig()

	return &Config{
		Token:   viper.GetString("token"),
		Project: viper.GetString("project"),
		Debug:   viper.GetBool("debug"),
		JSON:    viper.GetBool("json"),
		CI:      viper.GetBool("ci"),
	}
}

// UserConfigDir returns the OS-appropriate config directory.
func UserConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(os.Getenv("APPDATA"), "buildshare")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "buildshare")
	}
}

// CredentialsFilePath returns the path to the stored credentials file.
func CredentialsFilePath() string {
	return filepath.Join(UserConfigDir(), "credentials")
}

// ProjectConfigPath returns the path for a project-level config file.
func ProjectConfigPath() string {
	return ".buildshare.yaml"
}
