package probe

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	// Minimum time for first keyframe to be considered valid (seconds)
	minParsedKeyframeTime = 5.0
	// Minimum number of keyframes before considering extraction complete
	minParsedKeyframeCount = 3
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ExtractKeyframes extracts keyframe timestamps from a video file using ffprobe
// Uses packet inspection with keyframe flags to build a precise list
func (mp MediaProbe) ExtractKeyframes(ctx context.Context, videoPath string, videoIdx int) ([]float64, error) {
	if mp.Tools == nil {
		return nil, utils.ErrFFProbeUnavailable
	}

	ffprobePath := mp.Tools.FFProbe

	// Run ffprobe to get packet information with keyframe flags
	// Get all packets and filter for keyframes
	cmd := exec.CommandContext(ctx,
		ffprobePath,
		"-loglevel", "error",
		"-select_streams", fmt.Sprintf("V:%d", videoIdx),
		"-show_entries", "packet=pts_time,flags",
		// Some AVI files don't have pts, we use this to ask ffmpeg to generate them
		"-fflags", "+genpts",
		"-of", "csv=print_section=0",
		videoPath,
	)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start ffprobe: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	keyframes := make([]float64, 0, 1000)

	// Process each packet line
	for scanner.Scan() {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse CSV format: pts_time,flags
		parts := strings.Split(line, ",")
		if len(parts) < 2 {
			continue
		}

		pts, flags := parts[0], parts[1]

		// Skip if no valid timestamp (can happen with empty packets)
		if pts == "N/A" {
			break
		}

		// Only process keyframes (flags start with 'K')
		if len(flags) == 0 || flags[0] != 'K' {
			continue
		}

		// Parse timestamp
		timestamp, err := strconv.ParseFloat(pts, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp %s: %w", pts, err)
		}

		keyframes = append(keyframes, timestamp)
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("failed to probe video: %w", err)
	}

	// Check for scanner errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read ffprobe output: %w", err)
	}

	// Validate we have enough keyframes
	if len(keyframes) < minParsedKeyframeCount {
		return nil, fmt.Errorf("insufficient keyframes found: %d (minimum: %d)", len(keyframes), minParsedKeyframeCount)
	}

	// Check that we have keyframes after the minimum time threshold
	validKeyframes := 0
	for _, kf := range keyframes {
		if kf >= minParsedKeyframeTime {
			validKeyframes++
		}
	}

	if validKeyframes < minParsedKeyframeCount {
		return nil, fmt.Errorf("insufficient keyframes after %v seconds: %d (minimum: %d)",
			minParsedKeyframeTime, validKeyframes, minParsedKeyframeCount)
	}

	return keyframes, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ExtractKeyframesForVideo extracts keyframes for the first video stream in a file
// This is a convenience wrapper that always uses the first video stream (V:0)
func (mp MediaProbe) ExtractKeyframesForVideo(ctx context.Context, videoPath string) ([]float64, error) {
	return mp.ExtractKeyframes(ctx, videoPath, 0)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ValidateKeyframes validates that keyframes are in ascending order and reasonable
func ValidateKeyframes(keyframes []float64) error {
	if len(keyframes) == 0 {
		return fmt.Errorf("no keyframes provided")
	}

	// Check for ascending order
	for i := 1; i < len(keyframes); i++ {
		if keyframes[i] <= keyframes[i-1] {
			return fmt.Errorf("keyframes not in ascending order: %f <= %f at indices %d, %d",
				keyframes[i], keyframes[i-1], i, i-1)
		}
	}

	// Check for reasonable values (non-negative, not too large)
	for i, kf := range keyframes {
		if kf < 0 {
			return fmt.Errorf("negative keyframe timestamp at index %d: %f", i, kf)
		}
		if kf > 86400 { // More than 24 hours seems unreasonable
			return fmt.Errorf("unreasonably large keyframe timestamp at index %d: %f", i, kf)
		}
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetSegmentCount returns the number of segments that would be generated from keyframes
func GetSegmentCount(keyframes []float64) int {
	if len(keyframes) == 0 {
		return 0
	}
	return len(keyframes)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetSegmentDuration returns the duration of a specific segment based on keyframes
func GetSegmentDuration(keyframes []float64, segmentIndex int) float64 {
	if segmentIndex < 0 || segmentIndex >= len(keyframes) {
		return 0
	}

	// If this is the last segment, we can't determine duration without total video duration
	// For now, return 0 for the last segment (caller should handle this)
	if segmentIndex == len(keyframes)-1 {
		return 0
	}

	// Duration is the difference between this keyframe and the next
	return keyframes[segmentIndex+1] - keyframes[segmentIndex]
}
