package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/media"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Top-level summary for a single-media asset
type MediaInfo struct {
	DurationSec int
	File        ContainerInfo
	Video       VideoStream
	Audio       *AudioStream
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Container (file) facts
type ContainerInfo struct {
	Container  string // e.g. "mov,mp4,m4a,3gp,3g2,mj2"
	MIMEType   string // e.g. "video/mp4"
	SizeBytes  int64
	OverallBPS int // overall bitrate in bps
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Video stream only
type VideoStream struct {
	Codec  string // "h264", "hevc", "av1"
	Width  int
	Height int
	FPSNum int // avg_frame_rate numerator
	FPSDen int // avg_frame_rate denominator
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Audio stream
//
// TODO: support multiple tracks
type AudioStream struct {
	Language      string // "eng" / "und"
	Codec         string // "aac", "eac3", ...
	Profile       string // "LC", "Dolby Digital Plus"
	Channels      int    // 2, 6, etc. (normalize from layout if missing)
	ChannelLayout string // "stereo", "5.1"
	SampleRate    int    // Hz
	BitRate       int    // bps (may be 0 if unknown)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type MediaProbe struct {
	Tools *media.Tools
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ProbeVideo uses ffprobe to extract metadata from a video file
// Returns MediaInfo, video stream index, and error
func (mp MediaProbe) ProbeVideo(ctx context.Context, path string) (*MediaInfo, int, error) {
	if mp.Tools == nil {
		return nil, -1, fmt.Errorf("ffprobe unavailable: %w", utils.ErrFFProbeUnavailable)
	}

	ffprobePath := mp.Tools.FFProbe

	entries := []string{
		// format (container/file)
		"format=format_name,filename,size,bit_rate,duration",
		// common stream fields
		"stream=index,codec_type,codec_name,profile,bit_rate",
		// video
		"stream=width,height,avg_frame_rate,duration",
		// audio
		"stream=channels,channel_layout,sample_rate",
		// selection helpers
		"stream=disposition=default",
		"stream=tags=language",
	}

	cmd := exec.CommandContext(ctx,
		ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		"-show_entries", strings.Join(entries, ","),
		path,
	)

	out, err := cmd.Output()
	if err != nil {
		return nil, -1, fmt.Errorf("error running ffprobe: %w", err)
	}

	var p probeOutput
	if err := json.Unmarshal(out, &p); err != nil {
		return nil, -1, fmt.Errorf("parse: %w", err)
	}

	// pick first video stream with dimensions
	var v *stream
	var videoStreamIndex int = -1
	for i := range p.Streams {
		s := &p.Streams[i]
		if s.CodecType == "video" && s.Width > 0 && s.Height > 0 {
			v = s
			videoStreamIndex = s.Index
			break
		}
	}
	if v == nil {
		return nil, -1, fmt.Errorf("no video stream")
	}

	// pick audio: default if present, else first audio
	var a *stream
	for i := range p.Streams {
		s := &p.Streams[i]
		if s.CodecType == "audio" && s.Disposition.Default == 1 {
			a = s
			break
		}
	}
	if a == nil {
		for i := range p.Streams {
			s := &p.Streams[i]
			if s.CodecType == "audio" {
				a = s
				break
			}
		}
	}

	// duration (prefer stream, fallback to container)
	durStr := v.Duration
	if durStr == "" {
		durStr = p.Format.Duration
	}
	durF, _ := strconv.ParseFloat(durStr, 64)
	durationSec := int(math.Round(durF))

	// fps from video stream
	fpsN, fpsD := parseFPS(v.AvgFrameRate)

	info := &MediaInfo{
		DurationSec: durationSec,
		File: ContainerInfo{
			Container:  p.Format.FormatName,
			MIMEType:   guessMIME(p.Format.FormatName, p.Format.Filename),
			SizeBytes:  parseInt64(p.Format.Size),
			OverallBPS: parseInt(p.Format.BitRate),
		},
		Video: VideoStream{
			Codec:  v.CodecName,
			Width:  v.Width,
			Height: v.Height,
			FPSNum: fpsN,
			FPSDen: fpsD,
		},
		Audio: nil,
	}

	if a != nil {
		sr := parseInt(a.SampleRate)
		info.Audio = &AudioStream{
			Language:      strings.ToLower(a.Tags["language"]),
			Codec:         a.CodecName,
			Profile:       a.Profile,
			Channels:      normalizeChannels(a.Channels, a.ChannelLayout),
			ChannelLayout: a.ChannelLayout,
			SampleRate:    sr,
			BitRate:       parseInt(a.BitRate),
		}
	}

	return info, videoStreamIndex, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (v VideoStream) FPS() float64 {
	if v.FPSDen == 0 {
		return 0
	}
	return float64(v.FPSNum) / float64(v.FPSDen)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// resolutionLabel returns a human-readable label for the video resolution
func (v VideoStream) ResolutionLabel() string {
	h := v.Height
	switch {
	case h >= 4320:
		return "8K"
	case h >= 2160:
		return "4K"
	case h >= 1440:
		return "1440p"
	case h >= 1080:
		return "1080p"
	case h >= 720:
		return "720p"
	case h >= 480:
		return "480p"
	default:
		return fmt.Sprintf("%dp", h)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
func parseInt(s string) int { i, _ := strconv.Atoi(s); return i }

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func parseInt64(s string) int64 { i, _ := strconv.ParseInt(s, 10, 64); return i }

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func parseFPS(r string) (n, d int) {
	parts := strings.Split(r, "/")
	if len(parts) == 2 {
		n, _ = strconv.Atoi(parts[0])
		d, _ = strconv.Atoi(parts[1])
	}
	return
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func guessMIME(formatName, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".m4v", ".mov":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".mkv":
		return "video/x-matroska"
	}
	if strings.Contains(formatName, "matroska") {
		return "video/x-matroska"
	}
	if strings.Contains(formatName, "webm") {
		return "video/webm"
	}
	if strings.Contains(formatName, "mp4") || strings.Contains(formatName, "mov") {
		return "video/mp4"
	}
	return "application/octet-stream"
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func normalizeChannels(ch int, layout string) int {
	if ch > 0 {
		return ch
	}
	switch strings.ToLower(layout) {
	case "mono":
		return 1
	case "stereo", "2.0":
		return 2
	case "5.1":
		return 6
	case "7.1":
		return 8
	default:
		return ch
	}
}
