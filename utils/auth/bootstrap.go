package auth

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/geerew/off-course/utils/security"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// BootstrapToken represents a bootstrap token for initial admin setup
type BootstrapToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GenerateBootstrapToken creates a bootstrap token file for initial admin setup
func GenerateBootstrapToken(dataDir string, appFs afero.Fs) (*BootstrapToken, error) {
	token := security.RandomString(32)

	// Create bootstrap token
	bootstrapToken := &BootstrapToken{
		Token:     token,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		CreatedAt: time.Now(),
	}

	// Write token to file using appFs
	tokenPath := filepath.Join(dataDir, ".bootstrap-token")
	tokenData, err := json.Marshal(bootstrapToken)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := afero.WriteFile(appFs, tokenPath, tokenData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write token file: %w", err)
	}

	return bootstrapToken, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ValidateBootstrapToken validates a bootstrap token and returns the token data
func ValidateBootstrapToken(token, dataDir string, appFs afero.Fs) (*BootstrapToken, error) {
	tokenPath := filepath.Join(dataDir, ".bootstrap-token")

	// Check if token file exists
	if _, err := appFs.Stat(tokenPath); err != nil {
		return nil, fmt.Errorf("bootstrap token file not found")
	}

	// Read token file
	tokenData, err := afero.ReadFile(appFs, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Parse token
	var bootstrapToken BootstrapToken
	if err := json.Unmarshal(tokenData, &bootstrapToken); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate token
	if bootstrapToken.Token != token {
		return nil, fmt.Errorf("invalid token")
	}

	// Check expiration
	if time.Now().After(bootstrapToken.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return &bootstrapToken, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteBootstrapToken removes the bootstrap token file
func DeleteBootstrapToken(dataDir string, appFs afero.Fs) error {
	tokenPath := filepath.Join(dataDir, ".bootstrap-token")

	if err := appFs.Remove(tokenPath); err != nil {
		// Check if the error is because the file doesn't exist
		if _, statErr := appFs.Stat(tokenPath); statErr != nil {
			// File doesn't exist, that's fine
			return nil
		}
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}
