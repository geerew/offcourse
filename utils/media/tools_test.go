package media

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func toolsAvailable(t *testing.T) {
	t.Helper()

	_, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed; skipping test")
	}

	_, err = exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe not installed; skipping test")
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully resolving ffmpeg and ffprobe on PATH
func TestNewTools(t *testing.T) {
	toolsAvailable(t)

	tools, err := NewTools()
	require.NoError(t, err)
	require.NotNil(t, tools)
	require.NotEmpty(t, tools.FFmpeg)
	require.NotEmpty(t, tools.FFProbe)
}
