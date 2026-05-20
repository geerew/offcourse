package auth

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestGenerateBootstrapToken(t *testing.T) {
	appFs := afero.NewMemMapFs()
	tempDir := "/test-data"

	// Test successfully generating a token
	t.Run("simple", func(t *testing.T) {
		token, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)
		require.NotNil(t, token)
		require.NotEmpty(t, token.Token)
		require.GreaterOrEqual(t, len(token.Token), 32)
		require.True(t, time.Now().Before(token.ExpiresAt))
		require.True(t, time.Now().After(token.CreatedAt))
		require.True(t, token.ExpiresAt.After(token.CreatedAt))
		require.InDelta(t, (5 * time.Minute).Seconds(), token.ExpiresAt.Sub(token.CreatedAt).Seconds(), 2)

		tokenPath := filepath.Join(tempDir, ".bootstrap-token")
		exists, err := afero.Exists(appFs, tokenPath)
		require.NoError(t, err)
		require.True(t, exists)

		storedData, err := afero.ReadFile(appFs, tokenPath)
		require.NoError(t, err)
		var stored BootstrapToken
		require.NoError(t, json.Unmarshal(storedData, &stored))
		require.Equal(t, token.Token, stored.Token)
		require.True(t, token.ExpiresAt.Equal(stored.ExpiresAt))
		require.True(t, token.CreatedAt.Equal(stored.CreatedAt))
	})

	// Test successfully overwriting an existing bootstrap token file
	t.Run("overwrite", func(t *testing.T) {
		first, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)

		second, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)
		require.NotEqual(t, first.Token, second.Token)

		err = ValidateBootstrapToken(second.Token, tempDir, appFs)
		require.NoError(t, err)

		err = ValidateBootstrapToken(first.Token, tempDir, appFs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid token")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestValidateBootstrapToken(t *testing.T) {
	appFs := afero.NewMemMapFs()
	tempDir := "/test-data"

	// Test successfully validating a matching non-expired token
	t.Run("valid token", func(t *testing.T) {
		original, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)

		err = ValidateBootstrapToken(original.Token, tempDir, appFs)
		require.NoError(t, err)
	})

	// Test error when an invalid token is provided
	t.Run("invalid token", func(t *testing.T) {
		_, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)

		err = ValidateBootstrapToken("invalid-token", tempDir, appFs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid token")
	})

	// Test error when an empty token is provided
	t.Run("empty token", func(t *testing.T) {
		_, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)

		err = ValidateBootstrapToken("", tempDir, appFs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid token")
	})

	// Test error when a token with the same length but wrong bytes is provided
	t.Run("wrong token same length", func(t *testing.T) {
		original, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)

		wrong := []byte(original.Token)
		wrong[0] ^= 0xff
		err = ValidateBootstrapToken(string(wrong), tempDir, appFs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid token")
	})

	// Test error when the token file does not exist
	t.Run("token file not found", func(t *testing.T) {
		err := ValidateBootstrapToken("any-token", "/non-existent-dir", appFs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "bootstrap token file not found")
	})

	// Test error due to an expired token
	t.Run("expired token", func(t *testing.T) {
		expiredToken := &BootstrapToken{
			Token:     "test-token-value-32-chars-ok!!",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
			CreatedAt: time.Now().Add(-2 * time.Hour),
		}

		tokenPath := filepath.Join(tempDir, ".bootstrap-token")
		tokenData, err := json.Marshal(expiredToken)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(appFs, tokenPath, tokenData, 0600))

		err = ValidateBootstrapToken(expiredToken.Token, tempDir, appFs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "token expired")
	})

	// Test error parsing an invalid JSON token
	t.Run("invalid token file JSON", func(t *testing.T) {
		tokenPath := filepath.Join(tempDir, ".bootstrap-token")
		require.NoError(t, afero.WriteFile(appFs, tokenPath, []byte(`{not json`), 0600))

		err := ValidateBootstrapToken("any-token", tempDir, appFs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse token")
	})

	// Test error when the token file is deleted after generation
	t.Run("token file deleted", func(t *testing.T) {
		original, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)
		require.NoError(t, DeleteBootstrapToken(tempDir, appFs))

		err = ValidateBootstrapToken(original.Token, tempDir, appFs)
		require.Error(t, err)
		require.Contains(t, err.Error(), "bootstrap token file not found")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDeleteBootstrapToken(t *testing.T) {
	appFs := afero.NewMemMapFs()
	tempDir := "/test-data"
	tokenPath := filepath.Join(tempDir, ".bootstrap-token")

	// Test successfully deleting an token file
	t.Run("existing file", func(t *testing.T) {
		_, err := GenerateBootstrapToken(tempDir, appFs)
		require.NoError(t, err)

		exists, err := afero.Exists(appFs, tokenPath)
		require.NoError(t, err)
		require.True(t, exists)

		err = DeleteBootstrapToken(tempDir, appFs)
		require.NoError(t, err)

		exists, err = afero.Exists(appFs, tokenPath)
		require.NoError(t, err)
		require.False(t, exists)
	})

	// Test no error when deleting a non-existent token file
	t.Run("missing file", func(t *testing.T) {
		err := DeleteBootstrapToken(tempDir, appFs)
		require.NoError(t, err)
	})
}
