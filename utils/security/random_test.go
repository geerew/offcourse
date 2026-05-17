package security

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully generating a random string using the default alphanumeric
// charset
func Test_RandomString(t *testing.T) {
	generated := make([]string, 0, 1000)
	reg := regexp.MustCompile(`[a-zA-Z0-9]+`)
	length := 10

	for i := 0; i < 1000; i++ {
		res := RandomString(length)
		require.Len(t, res, length)
		require.True(t, reg.MatchString(res))

		for _, str := range generated {
			require.NotEqual(t, res, str)
		}

		generated = append(generated, res)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_RandomStringWithAlphabet(t *testing.T) {
	// Test successfully generating a random string using a custom alphabet with only
	// digits and underscore
	t.Run("only digits and underscore", func(t *testing.T) {
		generated := make([]string, 0, 1000)
		length := 10
		alphabet := "0123456789_"
		reg := regexp.MustCompile(`[0-9_]+`)

		for j := 0; j < 1000; j++ {
			res := RandomStringWithAlphabet(length, alphabet)
			require.Len(t, res, length)
			require.True(t, reg.MatchString(res))

			for _, str := range generated {
				require.NotEqual(t, res, str)
			}

			generated = append(generated, res)
		}
	})

	// Test successfully generating a random string using a custom alphabet with only special
	// characters
	t.Run("special characters", func(t *testing.T) {
		generated := make([]string, 0, 1000)
		length := 10
		alphabet := "!@#$%^&*()"
		reg := regexp.MustCompile(`[\!\@\#\$\%\^\&\*\(\)]+`)

		for j := 0; j < 1000; j++ {
			res := RandomStringWithAlphabet(length, alphabet)
			require.Len(t, res, length)
			require.True(t, reg.MatchString(res))

			for _, str := range generated {
				require.NotEqual(t, res, str)
			}

			generated = append(generated, res)
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully generating a pseudorandom string using the default alphanumeric charset
func Test_PseudorandomString(t *testing.T) {
	generated := make([]string, 0, 1000)
	reg := regexp.MustCompile(`[a-zA-Z0-9]+`)
	length := 10

	for i := 0; i < 1000; i++ {
		res := PseudorandomString(length)
		require.Len(t, res, length)
		require.True(t, reg.MatchString(res))

		for _, str := range generated {
			require.NotEqual(t, res, str)
		}

		generated = append(generated, res)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_PseudorandomStringWithAlphabet(t *testing.T) {
	// Test successfully generating a pseudorandom string using a custom alphabet with only
	// digits and underscore
	t.Run("digits and underscore", func(t *testing.T) {
		generated := make([]string, 0, 1000)
		length := 10
		alphabet := "0123456789_"
		reg := regexp.MustCompile(`[0-9_]+`)

		for j := 0; j < 1000; j++ {
			res := PseudorandomStringWithAlphabet(length, alphabet)
			require.Len(t, res, length)
			require.True(t, reg.MatchString(res))

			for _, str := range generated {
				require.NotEqual(t, res, str)
			}

			generated = append(generated, res)
		}
	})

	// Test successfully generating a pseudorandom string using a custom alphabet with only special
	// characters
	t.Run("special characters", func(t *testing.T) {
		generated := make([]string, 0, 1000)
		length := 10
		alphabet := "!@#$%^&*()"
		reg := regexp.MustCompile(`[\!\@\#\$\%\^\&\*\(\)]+`)

		for j := 0; j < 1000; j++ {
			res := PseudorandomStringWithAlphabet(length, alphabet)
			require.Len(t, res, length)
			require.True(t, reg.MatchString(res))

			for _, str := range generated {
				require.NotEqual(t, res, str)
			}
			generated = append(generated, res)
		}
	})
}
