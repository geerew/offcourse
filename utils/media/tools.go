package media

import (
	"os/exec"

	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Tools holds the paths to ffmpeg and ffprobe executables
type Tools struct {
	FFmpeg  string
	FFProbe string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewTools resolves the paths to ffmpeg and ffprobe on the system PATH
//
// Errors when either executable is not found
func NewTools() (*Tools, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, utils.ErrFFmpegUnavailable
	}

	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, utils.ErrFFProbeUnavailable
	}

	return &Tools{
		FFmpeg:  ffmpegPath,
		FFProbe: ffprobePath,
	}, nil
}
