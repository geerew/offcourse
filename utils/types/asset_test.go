package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAsset_NewAsset(t *testing.T) {
	// Test successfully creating an AssetType from a valid extensions
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			ext      string
			expected AssetType
		}{
			// Video
			{"avi", AssetVideo},
			{"mkv", AssetVideo},
			{"flac", AssetVideo},
			{"mp4", AssetVideo},
			{"m4a", AssetVideo},
			{"mp3", AssetVideo},
			{"ogv", AssetVideo},
			{"ogm", AssetVideo},
			{"ogg", AssetVideo},
			{"oga", AssetVideo},
			{"opus", AssetVideo},
			{"webm", AssetVideo},
			{"wav", AssetVideo},
			// document
			{"pdf", AssetPDF},
			// markdown
			{"md", AssetMarkdown},
			// text
			{"txt", AssetText},
		}

		for _, tt := range tests {
			a, err := NewAsset(tt.ext)
			require.NoError(t, err)
			require.Equal(t, tt.expected, a)
		}
	})

	// Test erroring when an invalid extension is provided
	t.Run("error", func(t *testing.T) {
		_, err := NewAsset("test")
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid asset extension")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAsset_MustAsset(t *testing.T) {
	// Test successfully creating an AssetType from a valid extensions
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			ext      string
			expected AssetType
		}{
			{"mp4", AssetVideo},
			{"pdf", AssetPDF},
			{"md", AssetMarkdown},
			{"txt", AssetText},
		}

		for _, tt := range tests {
			t.Run(tt.ext, func(t *testing.T) {
				result := MustAsset(tt.ext)
				require.Equal(t, tt.expected, result)
			})
		}
	})

	// Test panicking when an invalid extension is provided
	t.Run("panic on invalid", func(t *testing.T) {
		require.Panics(t, func() {
			MustAsset("invalid")
		}, "MustAsset should panic on invalid extension")

		require.Panics(t, func() {
			MustAsset("")
		}, "MustAsset should panic on empty extension")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAsset_Is(t *testing.T) {
	// Test successfully checking if an asset is a video
	t.Run("video", func(t *testing.T) {
		a, _ := NewAsset("mp4")
		require.True(t, a.IsVideo())
		require.True(t, a.IsValid())
	})

	// Test successfully checking if an asset is a PDF
	t.Run("pdf", func(t *testing.T) {
		a, _ := NewAsset("pdf")
		require.True(t, a.IsPDF())
		require.True(t, a.IsValid())
	})

	// Test successfully checking if an asset is a Markdown
	t.Run("markdown", func(t *testing.T) {
		a, _ := NewAsset("md")
		require.True(t, a.IsMarkdown())
		require.True(t, a.IsValid())
	})

	// Test successfully checking if an asset is a Text
	t.Run("text", func(t *testing.T) {
		a, _ := NewAsset("txt")
		require.True(t, a.IsText())
		require.True(t, a.IsValid())
	})

	// Test erroring when an invalid asset type is provided
	t.Run("invalid", func(t *testing.T) {
		invalid := AssetType("invalid")
		require.False(t, invalid.IsValid())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the string representation of an asset
func TestAsset_String(t *testing.T) {
	a, _ := NewAsset("mp4")
	require.Equal(t, "video", a.String())

	a, _ = NewAsset("pdf")
	require.Equal(t, "pdf", a.String())

	a, _ = NewAsset("md")
	require.Equal(t, "markdown", a.String())

	a, _ = NewAsset("txt")
	require.Equal(t, "text", a.String())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAsset_MarshalJSON(t *testing.T) {
	// Test successfully marshalling an asset to JSON
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
			hasError bool
		}{
			{"mp4", `"video"`, false},
			{"pdf", `"pdf"`, false},
			{"md", `"markdown"`, false},
			{"txt", `"text"`, false},
		}

		for _, tt := range tests {
			a, err := NewAsset(tt.input)
			require.NoError(t, err)

			res, err := a.MarshalJSON()
			if tt.hasError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, string(res))
			}
		}
	})

	// Test erroring when an invalid asset type is provided
	t.Run("error", func(t *testing.T) {
		invalid := AssetType("invalid")
		_, err := invalid.MarshalJSON()
		require.Error(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAsset_UnmarshalJSON(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			input    string
			expected AssetType
			err      string
		}{
			{`"video"`, AssetVideo, ""},
			{`"pdf"`, AssetPDF, ""},
			{`"markdown"`, AssetMarkdown, ""},
			{`"text"`, AssetText, ""},
		}

		for _, tt := range tests {
			var a AssetType
			err := a.UnmarshalJSON([]byte(tt.input))
			require.NoError(t, err)
			require.Equal(t, tt.expected, a)
		}
	})

	// Test erroring when an invalid JSON is provided
	t.Run("error", func(t *testing.T) {
		tests := []struct {
			input    string
			expected AssetType
			err      string
		}{
			// Invalid JSON
			{"", "", "unexpected end of JSON input"},
			{"xxx", "", "invalid character 'x' looking for beginning of value"},
			// Unknown asset types
			{`""`, "", "invalid asset type"},
			{`"bob"`, "", "invalid asset type"},
		}

		for _, tt := range tests {
			var a AssetType
			err := a.UnmarshalJSON([]byte(tt.input))

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.err)
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAsset_Value(t *testing.T) {
	// Test successfully getting the value of an asset
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"mp4", "video"},
			{"pdf", "pdf"},
			{"md", "markdown"},
			{"txt", "text"},
		}

		for _, tt := range tests {
			a, err := NewAsset(tt.input)
			require.NoError(t, err)

			res, err := a.Value()
			require.NoError(t, err)
			require.Equal(t, tt.expected, res)
		}
	})

	// Test erroring when an invalid asset type is provided
	t.Run("error", func(t *testing.T) {
		invalid := AssetType("invalid")
		_, err := invalid.Value()
		require.Error(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAsset_Scan(t *testing.T) {
	// Test successfully scanning an asset type
	t.Run("success", func(t *testing.T) {
		tests := []struct {
			value    any
			expected AssetType
		}{
			{"video", AssetVideo},
			{"pdf", AssetPDF},
			{"markdown", AssetMarkdown},
			{"text", AssetText},
		}

		for _, tt := range tests {
			var a AssetType

			err := a.Scan(tt.value)
			require.NoError(t, err)
			require.Equal(t, tt.expected, a)
		}
	})

	// Test erroring when an invalid asset type is provided
	t.Run("error", func(t *testing.T) {
		tests := []struct {
			value any
		}{
			{nil},
			{""},
			{"invalid"},
		}

		for _, tt := range tests {
			var a AssetType

			err := a.Scan(tt.value)
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid asset type")
		}
	})
}
