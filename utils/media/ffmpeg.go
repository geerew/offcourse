package media

import (
	"os/exec"

	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// FFmpeg holds the paths to ffmpeg and ffprobe executables
type FFmpeg struct {
	FFmpegPath  string
	FFProbePath string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewFFmpeg creates a new FFmpeg instance by resolving the paths to ffmpeg and
// ffprobe
//
// Errors when either executable is not found on the system PATH
func NewFFmpeg() (*FFmpeg, error) {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, utils.ErrFFmpegUnavailable
	}

	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, utils.ErrFFProbeUnavailable
	}

	return &FFmpeg{
		FFmpegPath:  ffmpegPath,
		FFProbePath: ffprobePath,
	}, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetFFmpegPath returns the resolved ffmpeg path
func (f *FFmpeg) GetFFmpegPath() string {
	return f.FFmpegPath
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetFFProbePath returns the resolved ffprobe path
func (f *FFmpeg) GetFFProbePath() string {
	return f.FFProbePath
}
