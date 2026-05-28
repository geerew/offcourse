package auth

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/geerew/off-course/utils/filesystem"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestGenerateRecoveryToken(t *testing.T) {
	fs := filesystem.New(afero.NewMemMapFs())
	dataDir := "/test-data"
	username := "admin"
	password := "newpassword123"

	// Test successfully generating a recovery token
	t.Run("simple", func(t *testing.T) {
		token, err := GenerateRecoveryToken(fs, username, password, dataDir)
		require.NoError(t, err)
		require.NotNil(t, token)
		require.Equal(t, username, token.Username)
		require.NotEmpty(t, token.Token)
		require.GreaterOrEqual(t, len(token.Token), 32)
		require.NotEmpty(t, token.PasswordHash)
		require.True(t, ComparePassword(token.PasswordHash, password))
		require.True(t, time.Now().Before(token.ExpiresAt))
		require.True(t, time.Now().After(token.CreatedAt))
		require.InDelta(t, (5 * time.Minute).Seconds(), token.ExpiresAt.Sub(token.CreatedAt).Seconds(), 2)

		tokenPath := filepath.Join(dataDir, ".recovery-token")
		exists, err := afero.Exists(fs, tokenPath)
		require.NoError(t, err)
		require.True(t, exists)

		storedData, err := afero.ReadFile(fs, tokenPath)
		require.NoError(t, err)
		var stored RecoveryToken
		require.NoError(t, json.Unmarshal(storedData, &stored))
		require.Equal(t, token.Username, stored.Username)
		require.Equal(t, token.Token, stored.Token)
		require.Equal(t, token.PasswordHash, stored.PasswordHash)
		require.True(t, token.ExpiresAt.Equal(stored.ExpiresAt))
		require.True(t, token.CreatedAt.Equal(stored.CreatedAt))
	})

	// Test successfully overwriting an existing recovery token
	t.Run("overwrite", func(t *testing.T) {
		first, err := GenerateRecoveryToken(fs, username, "first-pass123", dataDir)
		require.NoError(t, err)

		second, err := GenerateRecoveryToken(fs, username, "second-pass123", dataDir)
		require.NoError(t, err)
		require.NotEqual(t, first.Token, second.Token)
		require.NotEqual(t, first.PasswordHash, second.PasswordHash)

		validated, err := ValidateRecoveryToken(fs, second.Token, dataDir)
		require.NoError(t, err)
		require.True(t, ComparePassword(validated.PasswordHash, "second-pass123"))

		_, err = ValidateRecoveryToken(fs, first.Token, dataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid token")
	})

	// Test error when the password is longer than bcrypt allows
	t.Run("password longer than 72 bytes returns error", func(t *testing.T) {
		_, err := GenerateRecoveryToken(fs, username, strings.Repeat("a", 73), dataDir)
		require.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestValidateRecoveryToken(t *testing.T) {
	fs := filesystem.New(afero.NewMemMapFs())
	dataDir := "/test-data"
	username := "admin"
	password := "newpassword123"

	// Test successfully validating a token
	t.Run("valid token", func(t *testing.T) {
		original, err := GenerateRecoveryToken(fs, username, password, dataDir)
		require.NoError(t, err)

		validated, err := ValidateRecoveryToken(fs, original.Token, dataDir)
		require.NoError(t, err)
		require.Equal(t, original.Username, validated.Username)
		require.Equal(t, original.Token, validated.Token)
		require.Equal(t, original.PasswordHash, validated.PasswordHash)
		require.True(t, ComparePassword(validated.PasswordHash, password))
	})

	// Test error when an invalid token is provided
	t.Run("invalid token", func(t *testing.T) {
		_, err := GenerateRecoveryToken(fs, username, password, dataDir)
		require.NoError(t, err)

		_, err = ValidateRecoveryToken(fs, "invalid-token", dataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid token")
	})

	// Test error when an empty token is provided
	t.Run("empty token", func(t *testing.T) {
		_, err := GenerateRecoveryToken(fs, username, password, dataDir)
		require.NoError(t, err)

		_, err = ValidateRecoveryToken(fs, "", dataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid token")
	})

	// Test error when a token with the same length but wrong bytes is provided
	t.Run("wrong token same length", func(t *testing.T) {
		original, err := GenerateRecoveryToken(fs, username, password, dataDir)
		require.NoError(t, err)

		wrong := []byte(original.Token)
		wrong[0] ^= 0xff
		_, err = ValidateRecoveryToken(fs, string(wrong), dataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid token")
	})

	// Test error when the token file does not exist
	t.Run("token file not found", func(t *testing.T) {
		_, err := ValidateRecoveryToken(fs, "any-token", "/non-existent-dir")
		require.Error(t, err)
		require.Contains(t, err.Error(), "recovery token file not found")
	})

	// Test error when a token has expired
	t.Run("expired token", func(t *testing.T) {
		expired := &RecoveryToken{
			Username:     username,
			PasswordHash: "hash-not-used-for-expiry-test",
			Token:        "test-token-value-32-chars-ok!!",
			ExpiresAt:    time.Now().Add(-1 * time.Hour),
			CreatedAt:    time.Now().Add(-2 * time.Hour),
		}

		tokenPath := filepath.Join(dataDir, ".recovery-token")
		tokenData, err := json.Marshal(expired)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(fs, tokenPath, tokenData, 0600))

		_, err = ValidateRecoveryToken(fs, expired.Token, dataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "token expired")
	})

	// Test error when the token file is not valid JSON
	t.Run("invalid token file JSON", func(t *testing.T) {
		tokenPath := filepath.Join(dataDir, ".recovery-token")
		require.NoError(t, afero.WriteFile(fs, tokenPath, []byte(`{not json`), 0600))

		_, err := ValidateRecoveryToken(fs, "any-token", dataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse token")
	})

	// Test error when the token file is deleted after generation
	t.Run("token file deleted", func(t *testing.T) {
		original, err := GenerateRecoveryToken(fs, username, password, dataDir)
		require.NoError(t, err)
		require.NoError(t, DeleteRecoveryToken(fs, dataDir))

		_, err = ValidateRecoveryToken(fs, original.Token, dataDir)
		require.Error(t, err)
		require.Contains(t, err.Error(), "recovery token file not found")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDeleteRecoveryToken(t *testing.T) {
	fs := filesystem.New(afero.NewMemMapFs())
	dataDir := "/test-data"
	tokenPath := filepath.Join(dataDir, ".recovery-token")

	// Test successfully deleting an existing token file
	t.Run("existing file", func(t *testing.T) {
		_, err := GenerateRecoveryToken(fs, "admin", "password123", dataDir)
		require.NoError(t, err)

		exists, err := afero.Exists(fs, tokenPath)
		require.NoError(t, err)
		require.True(t, exists)

		err = DeleteRecoveryToken(fs, dataDir)
		require.NoError(t, err)

		exists, err = afero.Exists(fs, tokenPath)
		require.NoError(t, err)
		require.False(t, exists)
	})

	// Test no error when deleting a non-existent token file
	t.Run("missing file", func(t *testing.T) {
		err := DeleteRecoveryToken(fs, dataDir)
		require.NoError(t, err)
	})
}
