package auth

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestGeneratePassword(t *testing.T) {
	// Test successfully generating a bcrypt hash and comparing it to the original password
	t.Run("round trip for typical password", func(t *testing.T) {
		hash, err := GeneratePassword("correct horse battery staple")
		require.NoError(t, err)
		require.NotEmpty(t, hash)

		require.True(t, ComparePassword(hash, "correct horse battery staple"))
	})

	// Test each call uses a new salt so hashes differ
	t.Run("unique salt per hash", func(t *testing.T) {
		a, err := GeneratePassword("same")
		require.NoError(t, err)

		b, err := GeneratePassword("same")
		require.NoError(t, err)

		require.NotEqual(t, a, b)
		require.True(t, ComparePassword(a, "same"))
		require.True(t, ComparePassword(b, "same"))
	})

	// Test bcrypt rejects passwords longer than 72 bytes
	t.Run("password longer than 72 bytes returns error", func(t *testing.T) {
		_, err := GeneratePassword(strings.Repeat("a", 73))
		require.ErrorIs(t, err, bcrypt.ErrPasswordTooLong)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestComparePassword(t *testing.T) {
	// Test successfully rejecting the wrong plaintext password
	t.Run("wrong password", func(t *testing.T) {
		hash, err := GeneratePassword("secret")
		require.NoError(t, err)

		require.False(t, ComparePassword(hash, "not-secret"))
	})

	// Test successfully rejecting a non-bcrypt hash
	t.Run("invalid hash", func(t *testing.T) {
		require.False(t, ComparePassword("not-a-bcrypt-hash", "secret"))
	})
}
