package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/security"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RecoveryToken represents a recovery token for admin password reset
type RecoveryToken struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Token        string    `json:"token"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GenerateRecoveryToken creates a recovery token file for admin password reset
func GenerateRecoveryToken(appFs *appfs.AppFs, username, password, dataDir string) (*RecoveryToken, error) {
	token := security.RandomString(32)

	// Create recovery token
	recoveryToken := &RecoveryToken{
		Username:     username,
		PasswordHash: GeneratePassword(password),
		Token:        token,
		ExpiresAt:    time.Now().Add(5 * time.Minute),
		CreatedAt:    time.Now(),
	}

	// Write token to file
	tokenPath := filepath.Join(dataDir, ".recovery-token")
	tokenData, err := json.Marshal(recoveryToken)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := afero.WriteFile(appFs.Fs, tokenPath, tokenData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write token file: %w", err)
	}

	return recoveryToken, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ValidateRecoveryToken validates a recovery token and returns the token data
func ValidateRecoveryToken(appFs *appfs.AppFs, token, dataDir string) (*RecoveryToken, error) {
	tokenPath := filepath.Join(dataDir, ".recovery-token")

	// Check if token file exists
	if _, err := appFs.Fs.Stat(tokenPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("recovery token file not found")
	}

	// Read token file
	tokenData, err := afero.ReadFile(appFs.Fs, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Parse token
	var recoveryToken RecoveryToken
	if err := json.Unmarshal(tokenData, &recoveryToken); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Validate token
	if recoveryToken.Token != token {
		return nil, fmt.Errorf("invalid token")
	}

	// Check expiration
	if time.Now().After(recoveryToken.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return &recoveryToken, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteRecoveryToken removes the recovery token file
func DeleteRecoveryToken(appFs *appfs.AppFs, dataDir string) error {
	tokenPath := filepath.Join(dataDir, ".recovery-token")

	if err := appFs.Fs.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}
