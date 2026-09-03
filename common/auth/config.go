package auth

import (
	"github.com/open-cli-collective/google-cli/common/config"
)

// Re-export constants for backward compatibility
const (
	// CredentialsFile is the name of the OAuth credentials file
	// Deprecated: Use config.CredentialsFile instead
	CredentialsFile = config.CredentialsFile
	// TokenFile is the name of the OAuth token file (fallback storage)
	// Deprecated: Use config.TokenFile instead
	TokenFile = config.TokenFile
)

// GetConfigDir returns the configuration directory path, creating it if needed.
// Deprecated: Use config.GetConfigDir() instead
func GetConfigDir() (string, error) {
	return config.GetConfigDir()
}

// GetCredentialsPath returns the full path to credentials.json
// Deprecated: Use config.GetCredentialsPath() instead
func GetCredentialsPath() (string, error) {
	return config.GetCredentialsPath()
}

// GetTokenPath returns the full path to token.json (fallback storage)
// Deprecated: Use config.GetTokenPath() instead
func GetTokenPath() (string, error) {
	return config.GetTokenPath()
}

// ShortenPath replaces the home directory prefix with ~ for display purposes.
// Deprecated: Use config.ShortenPath() instead
func ShortenPath(path string) string {
	return config.ShortenPath(path)
}
