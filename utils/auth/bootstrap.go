package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"github.com/geerew/off-course/utils/security"
	"github.com/spf13/afero"
)

// Bootstrap runs when the application has no admin user yet (first start).
//
// Flow:
//  1. The app writes a short-lived secret to .bootstrap-token in the data directory and prints a bootstrap URL (token in the path).
//  2. An operator opens that URL and submits username/password; POST /api/auth/bootstrap/:token validates the token and creates the first admin via the API/DAO.
//  3. On success the token file is removed and the app is marked bootstrapped; further bootstrap attempts are rejected.
//
// The token file is the capability: only someone with access to the data directory (or the printed URL before expiry) can complete setup.

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// BootstrapToken represents a bootstrap token for initial admin setup
type BootstrapToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GenerateBootstrapToken creates a .bootstrap-token file in the data directory
func GenerateBootstrapToken(dataDir string, appFs afero.Fs) (*BootstrapToken, error) {
	token := security.RandomString(32)

	bootstrapToken := &BootstrapToken{
		Token:     token,
		ExpiresAt: time.Now().Add(5 * time.Minute),
		CreatedAt: time.Now(),
	}

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

// ValidateBootstrapToken checks the bootstrap token against the token in the .bootstrap-token
//
// It returns nil when the token is valid
func ValidateBootstrapToken(token, dataDir string, appFs afero.Fs) error {
	tokenPath := filepath.Join(dataDir, ".bootstrap-token")

	tokenData, err := afero.ReadFile(appFs, tokenPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("bootstrap token file not found")
		}

		return fmt.Errorf("failed to read token file: %w", err)
	}

	var bootstrapToken BootstrapToken
	if err := json.Unmarshal(tokenData, &bootstrapToken); err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if len(token) != len(bootstrapToken.Token) {
		return fmt.Errorf("invalid token")
	}

	if subtle.ConstantTimeCompare([]byte(token), []byte(bootstrapToken.Token)) != 1 {
		return fmt.Errorf("invalid token")
	}

	if time.Now().After(bootstrapToken.ExpiresAt) {
		return fmt.Errorf("token expired")
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteBootstrapToken removes the bootstrap token file
func DeleteBootstrapToken(dataDir string, appFs afero.Fs) error {
	tokenPath := filepath.Join(dataDir, ".bootstrap-token")

	if err := appFs.Remove(tokenPath); err != nil {
		if _, statErr := appFs.Stat(tokenPath); statErr != nil {
			// Ignore this error when the file doesn't exist
			return nil
		}

		return fmt.Errorf("failed to delete token file: %w", err)
	}

	return nil
}
