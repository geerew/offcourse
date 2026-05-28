package probe

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geerew/off-course/utils/media"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func ffprobeAvailable(t *testing.T) {
	t.Helper()
	_, err := exec.LookPath("ffprobe")
	if err != nil {
		t.Skip("ffprobe not installed; skipping test")
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestProbeVideo(t *testing.T) {
	ffprobeAvailable(t)

	t.Run("valid video", func(t *testing.T) {
		testVideo := filepath.Join("testdata", "sample.mp4")

		tools, err := media.NewTools()
		require.NoError(t, err)

		mp := MediaProbe{Tools: tools}
		info, videoIdx, err := mp.ProbeVideo(context.Background(), testVideo)
		require.NoError(t, err)
		require.NotNil(t, info)
		require.GreaterOrEqual(t, videoIdx, 0)

		// Duration: we generated a 5s sample
		require.Equal(t, 5, info.DurationSec)

		// Video stream facts
		require.Equal(t, 1280, info.Video.Width)
		require.Equal(t, 720, info.Video.Height)
		require.Equal(t, "h264", strings.ToLower(info.Video.Codec))
		require.Equal(t, 30, info.Video.FPSNum)
		require.Equal(t, 1, info.Video.FPSDen)

		// Container / file facts
		require.Equal(t, "video/mp4", info.File.MIMEType)
		require.Greater(t, info.File.SizeBytes, int64(0))
		require.Greater(t, info.File.OverallBPS, 0)

		// Audio (single selected track)
		require.NotNil(t, info.Audio)
		require.Equal(t, "aac", strings.ToLower(info.Audio.Codec))
		require.Equal(t, 48000, info.Audio.SampleRate)
		require.GreaterOrEqual(t, info.Audio.Channels, 1) // mono on the synthetic sample
	})

	t.Run("invalid video", func(t *testing.T) {
		tools, err := media.NewTools()
		require.NoError(t, err)

		mp := MediaProbe{Tools: tools}
		_, _, err = mp.ProbeVideo(context.Background(), "testdata/does_not_exist.mp4")
		require.Error(t, err)
		require.Contains(t, err.Error(), "error running ffprobe")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
