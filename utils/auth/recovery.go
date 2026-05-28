package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/geerew/off-course/utils/filesystem"
	"github.com/geerew/off-course/utils/security"
	"github.com/spf13/afero"
)

// Recovery resets a lost admin password when the operator has shell access to the data directory.
//
// Flow:
//  1. The admin reset-password CLI hashes the new password and writes .recovery-token (username, password hash, short-lived secret).
//  2. The CLI POSTs only the secret to POST /api/admin/recovery on the running server.
//  3. The API validates the token and updates the admin row via the DAO (the running app owns DB access); the token file is then deleted.
//
// Password material never goes over HTTP—only the one-time token. The API endpoint is unauthenticated but useless without a valid .recovery-token on disk.

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RecoveryToken represents a recovery token for password reset
type RecoveryToken struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Token        string    `json:"token"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GenerateRecoveryToken creates a .recovery-token file in the data directory
func GenerateRecoveryToken(fs *filesystem.FS, username, password, dataDir string) (*RecoveryToken, error) {
	token := security.RandomString(32)

	hash, err := GeneratePassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	recoveryToken := &RecoveryToken{
		Username:     username,
		PasswordHash: hash,
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

	if err := afero.WriteFile(fs, tokenPath, tokenData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write token file: %w", err)
	}

	return recoveryToken, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ValidateRecoveryToken checks the recovery token against the token in the .recovery-token
//
// It returns the token data when the token is valid, else it returns an error
func ValidateRecoveryToken(fs *filesystem.FS, token, dataDir string) (*RecoveryToken, error) {
	tokenPath := filepath.Join(dataDir, ".recovery-token")

	if _, err := fs.Stat(tokenPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("recovery token file not found")
	}

	tokenData, err := afero.ReadFile(fs, tokenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	var recoveryToken RecoveryToken
	if err := json.Unmarshal(tokenData, &recoveryToken); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if recoveryToken.Token != token {
		return nil, fmt.Errorf("invalid token")
	}

	if time.Now().After(recoveryToken.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	return &recoveryToken, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteRecoveryToken deletes the .recovery-token file
func DeleteRecoveryToken(fs *filesystem.FS, dataDir string) error {
	tokenPath := filepath.Join(dataDir, ".recovery-token")

	if err := fs.Remove(tokenPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}
