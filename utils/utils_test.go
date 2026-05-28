package utils

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DecodeString(t *testing.T) {
	// Test successfully decoding a valid Base64-encoded string
	t.Run("success", func(t *testing.T) {
		res, err := DecodeString("JTJGdGVzdCUyRmRhdGE=")
		require.NoError(t, err)
		require.Equal(t, "/test/data", res)
	})

	// Test no error when decoding an empty string
	t.Run("empty", func(t *testing.T) {
		res, err := DecodeString("")
		require.NoError(t, err)
		require.Equal(t, "", res)
	})

	// Test erroring when a invalid Base64 string is provided
	t.Run("decode error", func(t *testing.T) {
		res, err := DecodeString("`")
		require.EqualError(t, err, "failed to decode path")
		require.Empty(t, res)
	})

	// Test erroring when a invalid URL-encoded string is provided
	t.Run("unescape error", func(t *testing.T) {
		res, err := DecodeString("dGVzdCUyMDElMiUyNiUyMHRlc3QlMjAy")
		require.EqualError(t, err, "failed to unescape path")
		require.Empty(t, res)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_EncodeString(t *testing.T) {
	// Test encoding an empty string
	t.Run("empty", func(t *testing.T) {
		res := EncodeString("")
		require.Equal(t, "", res)
	})

	// Test encoding a valid string
	t.Run("success", func(t *testing.T) {
		res := EncodeString("/test/data")
		require.Equal(t, "JTJGdGVzdCUyRmRhdGE=", res)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully normalizing a Windows drive path
//
// These tests are skipped on non-Windows systems because the underlying function only
// runs on Windows systems
func Test_NormalizeWindowsDrive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific tests on non-Windows systems")
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"C:", "C:\\"},
		{"C:\\", "C:\\"},
		{"C:folder", "C:\\folder"},
		{"C:\\folder", "C:\\folder"},
	}

	for _, test := range tests {
		got := NormalizeWindowsDrive(test.input)
		require.Equal(t, test.expected, got)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting an environment variable
func Test_GetEnvOr(t *testing.T) {
	_ = os.Unsetenv("UTILS_TEST_ENV")
	require.Equal(t, "default", GetEnvOr("UTILS_TEST_ENV", "default"))

	t.Setenv("UTILS_TEST_ENV", "value")
	require.Equal(t, "value", GetEnvOr("UTILS_TEST_ENV", "default"))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_IsCard(t *testing.T) {
	// Test checking valid card filenames
	t.Run("valid", func(t *testing.T) {
		var tests = []string{
			"card.jpg",
			"card.jpeg",
			"card.png",
			"card.webp",
			"card.tiff",
		}

		for _, tt := range tests {
			require.True(t, IsCard(tt))
		}
	})

	// Test checking invalid card filenames
	t.Run("invalid", func(t *testing.T) {
		var tests = []string{
			"card",
			"1234",
			"1234.jpg",
			"jpg",
			"card.test.jpg",
			"card.txt",
		}

		for _, tt := range tests {
			require.False(t, IsCard(tt))
		}
	})

}
