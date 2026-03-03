// Package hls provides on-demand HLS transcoding and segmentation with
// adaptive prefetching, hardware acceleration, and per-stream tracking
package hls

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Head represents an encoding process
type Head struct {
	segment int32
	end     int32
	command *exec.Cmd
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeletedHead is a marker for a head that has been killed
var DeletedHead = Head{
	segment: -1,
	end:     -1,
	command: nil,
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Segment represents a single HLS segment
type Segment struct {
	channel chan struct{}
	encoder int
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Flags represents stream type flags
type Flags int32

const (
	AudioF   Flags = 1 << 0
	VideoF   Flags = 1 << 1
	Transmux Flags = 1 << 3
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// VideoKey uniquely identifies a video stream by index and quality
type VideoKey struct {
	idx     uint32
	quality Quality
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Streamer represents a stream interface for transcoding operations
type Streamer interface {
	getTranscodeArgs(segments string) []string
	getOutPath(encoderID int) string
	getFlags() Flags
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Stream represents a transcoding a single stream (audio or video) with multiple
// encoding heads
type Stream struct {
	streamer      Streamer
	streamWrapper *StreamWrapper
	keyframes     []float64
	segments      []Segment
	heads         []Head
	lock          sync.RWMutex
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initializeSegments creates and initializes segments based on keyframes
func (s *Stream) initializeSegments() {
	length := len(s.keyframes)
	s.segments = make([]Segment, length, max(length, 2000))
	for seg := range s.segments {
		s.segments[seg].channel = make(chan struct{})
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isSegmentReady checks if a segment is ready (non-blocking)
//
// Always lock before calling this
func (s *Stream) isSegmentReady(segment int32) bool {
	select {
	case <-s.segments[segment].channel:
		// If the channel returned, it means it was closed
		return true
	default:
		return false
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isSegmentTranscoding checks if a segment is currently being transcoded
func (s *Stream) isSegmentTranscoding(segment int32) bool {
	for _, head := range s.heads {
		if head.segment == segment {
			return true
		}
	}
	return false
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// run starts transcoding from the given segment
func (s *Stream) run(startSegment int32) error {
	// Start the transcode with adaptive buffer based on video length
	length := len(s.keyframes)

	// Calculate smart buffer size based on video duration
	videoDuration := float64(s.streamWrapper.Info.Duration)
	var bufferSegments int32

	if videoDuration <= 300 {
		bufferSegments = 15
	} else if videoDuration <= 600 {
		bufferSegments = 20
	} else {
		bufferSegments = 25
	}

	endSegment := min(startSegment+bufferSegments, int32(length))

	s.lock.Lock()

	// Stop at the first finished segment
	for i := startSegment; i < endSegment; i++ {
		if s.isSegmentReady(i) || s.isSegmentTranscoding(i) {
			endSegment = i
			break
		}
	}

	// A startSegment can be equal to or greater than the endSegment if the start
	// finished between checks
	if startSegment >= endSegment {
		s.lock.Unlock()
		return nil
	}

	encoderID := len(s.heads)
	s.heads = append(s.heads, Head{segment: startSegment, end: endSegment, command: nil})
	s.lock.Unlock()

	s.streamWrapper.config.Logger.Debug().
		Str("asset_id", s.streamWrapper.assetID).
		Str("path", s.streamWrapper.Info.Path).
		Int("encoder_id", encoderID).
		Int32("start_segment", startSegment).
		Int32("end_segment", endSegment).
		Int("total_segments", length).
		Msg("Starting transcode")

	// Calculate FFmpeg seek references for precise segment cutting
	startRef, endRef := s.calculateSeekReferences(startSegment, endSegment, int32(length))

	endPadding := int32(1)
	if endSegment == int32(length) {
		endPadding = 0
	}

	// Calculate the segments to transcode
	segments := s.keyframes[startSegment+1 : endSegment+endPadding]
	if len(segments) == 0 {
		// ffmpeg errors out if the segments are empty
		segments = []float64{9999999}
	}

	// Create output directory
	outPath := s.streamer.getOutPath(encoderID)
	err := s.streamWrapper.config.AppFs.Fs.MkdirAll(filepath.Dir(outPath), 0o755)
	if err != nil {
		s.streamWrapper.config.Logger.Error().
			Err(err).
			Str("asset_id", s.streamWrapper.assetID).
			Str("path", s.streamWrapper.Info.Path).
			Int("encoder_id", encoderID).
			Str("out_path", outPath).
			Msg("Failed to create output directory for transcoding")
		return err
	}

	// Build the FFmpeg arguments
	args := []string{
		"-nostats", "-hide_banner", "-loglevel", "warning",
	}

	// Add the hardware acceleration flags if the stream is a video
	if s.streamer.getFlags()&VideoF != 0 {
		args = append(args, s.streamWrapper.config.HwAccel.DecodeFlags...)
	}

	// Add -ss parameter when we are not at the beginning of the stream
	if startRef != 0 {
		// Required for video to force pre/post segment to work
		if s.streamer.getFlags()&VideoF != 0 {
			args = append(args, "-noaccurate_seek")
		}

		args = append(args, "-ss", fmt.Sprintf("%.6f", startRef))
	}

	// Add -to parameter when we don't want to go to the end of the stream
	if endRef > 0 {
		args = append(args, "-to", fmt.Sprintf("%.6f", endRef))
	}

	args = append(args,
		// If timestamps (PTS) are missing or messy after the seek/streamcopy, generate new PTS so
		// downstream muxers stay happy
		"-fflags", "+genpts",
		// Input file
		"-i", s.streamWrapper.Info.Path,
		// Ensure consistent behavior between software and hardware decoding
		"-start_at_zero",
		// Preserve input timestamps
		"-copyts",
		// Do not buffer at the muxer, instead write the packets as soon as possible
		"-muxdelay", "0",
	)

	// Add the transcoding arguments for the type of stream (audio or video)
	args = append(args, s.streamer.getTranscodeArgs(toSegmentStr(segments))...)

	args = append(args,
		//Uuse the segment muxer
		"-f", "segment",
		// Allow small timing variations for keyframe alignment
		"-segment_time_delta", "0.05",
		// Write each segment as a MPEG-TS file
		"-segment_format", "mpegts",
		// Explicit split times, relative to the seek point
		"-segment_times", strings.Join(utils.Map(segments, func(seg float64) string {
			return fmt.Sprintf("%.6f", seg-s.keyframes[startSegment])
		}), ","),
		// Wirte a flat list of segments
		"-segment_list_type", "flat",
		// Write the segment list to stdout, instead of to a file
		"-segment_list", "pipe:1",
		// The starting number of the segment to write to the output path
		"-segment_start_number", fmt.Sprint(startSegment),
		// The output path to write the segment to
		outPath,
	)

	// Run the FFmpeg command
	cmd := exec.Command("ffmpeg", args...)
	s.streamWrapper.config.Logger.Debug().
		Str("asset_id", s.streamWrapper.assetID).
		Str("path", s.streamWrapper.Info.Path).
		Int("encoder_id", encoderID).
		Str("command", strings.Join(cmd.Args, " ")).
		Msg("Running FFmpeg")

	// Set the stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		s.streamWrapper.config.Logger.Error().
			Err(err).
			Str("asset_id", s.streamWrapper.assetID).
			Str("path", s.streamWrapper.Info.Path).
			Int("encoder_id", encoderID).
			Msg("Failed to create stdout pipe for FFmpeg")
		return err
	}

	// Set the stderr pipe
	var stderr strings.Builder
	cmd.Stderr = &stderr

	// Start the command
	err = cmd.Start()
	if err != nil {
		s.streamWrapper.config.Logger.Error().
			Err(err).
			Str("asset_id", s.streamWrapper.assetID).
			Str("path", s.streamWrapper.Info.Path).
			Int("encoder_id", encoderID).
			Msg("Failed to start FFmpeg command")
		return err
	}

	// Store the command in the heads slice
	s.lock.Lock()
	s.heads[encoderID].command = cmd
	s.lock.Unlock()

	// Monitor FFmpeg output to track segment completion
	go func() {
		scanner := bufio.NewScanner(stdout)
		format := filepath.Base(outPath)
		shouldStop := false

		// Parse each line of FFmpeg output to get completed segment numbers
		for scanner.Scan() {
			var segment int32
			_, _ = fmt.Sscanf(scanner.Text(), format, &segment)

			// Skip segments before our start point (due to -f segment padding)
			if segment < startSegment {
				continue
			}

			s.lock.Lock()
			s.heads[encoderID].segment = segment

			// Determine stream type for logging
			streamType := "unknown"
			if s.streamer.getFlags()&AudioF != 0 {
				streamType = "audio"
			} else if s.streamer.getFlags()&VideoF != 0 {
				streamType = "video"
			}

			s.streamWrapper.config.Logger.Debug().
				Str("asset_id", s.streamWrapper.assetID).
				Str("path", s.streamWrapper.Info.Path).
				Str("stream_type", streamType).
				Int32("segment", segment).
				Int("encoder_id", encoderID).
				Msg("Segment is ready")

			// Check if this segment is already completed by another encoder
			if s.isSegmentReady(segment) {
				// Another encoder already completed this segment, stop this one
				cmd.Process.Signal(os.Interrupt)
				s.streamWrapper.config.Logger.Debug().
					Str("asset_id", s.streamWrapper.assetID).
					Str("path", s.streamWrapper.Info.Path).
					Str("stream_type", streamType).
					Int("encoder_id", encoderID).
					Int32("segment", segment).
					Msg("Stopping encoder because segment is already ready")
				shouldStop = true
			} else {
				// Mark this segment as completed by this encoder
				s.segments[segment].encoder = encoderID
				close(s.segments[segment].channel)

				// Check if we should stop encoding
				if segment == endSegment-1 {
					// Reached the end of our target range
					shouldStop = true
				} else if s.isSegmentReady(segment + 1) {
					// Next segment is already ready, stop to avoid duplicate work
					cmd.Process.Signal(os.Interrupt)
					s.streamWrapper.config.Logger.Debug().
						Str("asset_id", s.streamWrapper.assetID).
						Str("path", s.streamWrapper.Info.Path).
						Str("stream_type", streamType).
						Int("encoder_id", encoderID).
						Int32("segment", segment).
						Msg("Killing ffmpeg because next segment is ready")
					shouldStop = true
				}
			}

			s.lock.Unlock()

			if shouldStop {
				return
			}
		}

		// Handle any scanner errors
		if err := scanner.Err(); err != nil {
			s.streamWrapper.config.Logger.Error().
				Err(err).
				Str("asset_id", s.streamWrapper.assetID).
				Str("path", s.streamWrapper.Info.Path).
				Int("encoder_id", encoderID).
				Msg("Error reading stdout of ffmpeg")
		}
	}()

	// Wait for FFmpeg process to complete and clean up the encoder
	go func() {
		err := cmd.Wait()

		if exiterr, ok := err.(*exec.ExitError); ok && exiterr.ExitCode() == 255 {
			// FFmpeg was interrupted by us (normal termination)
			s.streamWrapper.config.Logger.Debug().
				Str("asset_id", s.streamWrapper.assetID).
				Str("path", s.streamWrapper.Info.Path).
				Int("encoder_id", encoderID).
				Msg("ffmpeg was killed by us")
		} else if err != nil {
			// FFmpeg encountered an error during execution
			s.streamWrapper.config.Logger.Error().
				Err(err).
				Str("asset_id", s.streamWrapper.assetID).
				Str("path", s.streamWrapper.Info.Path).
				Int("encoder_id", encoderID).
				Str("stderr", stderr.String()).
				Msg("ffmpeg occurred an error")
		} else {
			// FFmpeg completed successfully
			s.streamWrapper.config.Logger.Debug().
				Str("asset_id", s.streamWrapper.assetID).
				Str("path", s.streamWrapper.Info.Path).
				Int("encoder_id", encoderID).
				Msg("ffmpeg finished successfully")
		}

		s.lock.Lock()
		defer s.lock.Unlock()

		// Mark as deleted instead of removing to preserve encoder IDs for other heads
		s.heads[encoderID] = DeletedHead
	}()

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// calculateSeekReferences calculates the start and end seek references for FFmpeg
func (s *Stream) calculateSeekReferences(startSegment, endSegment, length int32) (float64, float64) {
	startRef := float64(0)
	endRef := float64(0)

	// Calculate start reference
	if startSegment != 0 {
		actualStartSegment := startSegment - 1

		if s.streamer.getFlags()&AudioF != 0 {
			// For audio, FFmpeg needs context before the starting point, without that it doesn't know what
			// to do and leaves ~100ms of silence
			startRef = s.keyframes[actualStartSegment]
		} else {
			// For video: FFmpeg's -ss parameter seeks to the keyframe before the specified time. To get
			// precise seeking, we specify a point slightly after the target keyframe
			if actualStartSegment+1 == int32(length) {
				startRef = (s.keyframes[actualStartSegment] + float64(s.streamWrapper.Info.Duration)) / 2
			} else {
				startRef = (s.keyframes[actualStartSegment] + s.keyframes[actualStartSegment+1]) / 2
			}
		}
	}

	// Calculate end reference
	if endSegment+1 < int32(length) {
		// Include extra padding and use -f segment for precise cutting
		endRef = s.keyframes[endSegment+1]

		// Adjust for seek offset when startRef is used
		if startRef > 0 {
			endRef += startRef - s.keyframes[startSegment-1]
		}
	}

	return startRef, endRef
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetIndex generates the HLS index playlist for the stream (video or audio)
func (s *Stream) GetIndex() (string, error) {
	index := `#EXTM3U
#EXT-X-VERSION:6
#EXT-X-TARGETDURATION:6
#EXT-X-MEDIA-SEQUENCE:0
#EXT-X-INDEPENDENT-SEGMENTS
`
	length := len(s.keyframes)

	for segment := int32(0); segment < int32(length)-1; segment++ {
		index += fmt.Sprintf("#EXTINF:%.6f\n", s.keyframes[segment+1]-s.keyframes[segment])
		index += fmt.Sprintf("segment-%d.ts\n", segment)
	}

	index += fmt.Sprintf("#EXTINF:%.6f\n", float64(s.streamWrapper.Info.Duration)-s.keyframes[length-1])
	index += fmt.Sprintf("segment-%d.ts\n", length-1)

	index += `#EXT-X-ENDLIST`

	return index, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetSegment retrieves a specific segment path, starting transcoding if needed
func (s *Stream) GetSegment(segment int32) (string, error) {
	s.lock.RLock()
	ready := s.isSegmentReady(segment)

	distance := 0.
	isScheduled := false

	// Determine the distance to the next encoder and if the segment is scheduled
	if !ready {
		distance = s.getMinEncoderDistance(segment)
		for _, head := range s.heads {
			if head.segment <= segment && segment < head.end {
				isScheduled = true
				break
			}
		}
	}

	readyChan := s.segments[segment].channel
	s.lock.RUnlock()

	if !ready {
		// Only start a new encode if there is too big a distance between the current encoder and the segment.
		if distance > 60 || !isScheduled {
			s.streamWrapper.config.Logger.Info().
				Str("asset_id", s.streamWrapper.assetID).
				Str("path", s.streamWrapper.Info.Path).
				Int32("segment", segment).
				Float64("distance", distance).
				Msg("Creating new head since closest head is far away")
			err := s.run(segment)
			if err != nil {
				s.streamWrapper.config.Logger.Error().
					Err(err).
					Str("asset_id", s.streamWrapper.assetID).
					Str("path", s.streamWrapper.Info.Path).
					Int32("segment", segment).
					Msg("Failed to start transcoding")
				return "", err
			}
		} else {
			s.streamWrapper.config.Logger.Debug().
				Str("asset_id", s.streamWrapper.assetID).
				Str("path", s.streamWrapper.Info.Path).
				Int32("segment", segment).
				Float64("distance", distance).
				Msg("Waiting for segment since encoder head is close")
		}

		select {
		case <-readyChan:
		case <-time.After(60 * time.Second):
			err := errors.New("could not retrieve the selected segment (timeout)")
			s.streamWrapper.config.Logger.Warn().
				Err(err).
				Str("asset_id", s.streamWrapper.assetID).
				Str("path", s.streamWrapper.Info.Path).
				Int32("segment", segment).
				Msg("Segment retrieval timeout")
			return "", err
		}
	}

	s.prepareNextSegments(segment)
	return fmt.Sprintf(s.streamer.getOutPath(s.segments[segment].encoder), segment), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// prepareNextSegments starts encoding future segments for video streams
func (s *Stream) prepareNextSegments(segment int32) {
	// Skip audio streams as they are cheap to encode on-demand
	if s.streamer.getFlags()&VideoF == 0 {
		return
	}

	s.lock.RLock()
	defer s.lock.RUnlock()

	for i := segment + 1; i <= min(segment+10, int32(len(s.segments)-1)); i++ {
		if s.isSegmentReady(i) {
			continue
		}

		// Skip if too close to active encoder (60s buffer + 5s per segment)
		if s.getMinEncoderDistance(i) < 60+(5*float64(i-segment)) {
			continue
		}

		s.streamWrapper.config.Logger.Debug().
			Str("asset_id", s.streamWrapper.assetID).
			Str("path", s.streamWrapper.Info.Path).
			Int32("segment", i).
			Msg("Creating new head for future segment")
		go s.run(i)

		return
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getMinEncoderDistance calculates the minimum distance to any active encoder
func (s *Stream) getMinEncoderDistance(segment int32) float64 {
	time := s.keyframes[segment]

	distances := utils.Map(s.heads, func(head Head) float64 {
		// Ignore killed heads or heads after the current time
		if head.segment < 0 || s.keyframes[head.segment] > time || segment >= head.end {
			return math.Inf(1)
		}

		return time - s.keyframes[head.segment]
	})

	if len(distances) == 0 {
		return math.Inf(1)
	}

	return slices.Min(distances)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Kill stops all encoding processes associated with the stream
func (s *Stream) Kill() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for id := range s.heads {
		s.KillHead(id)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// KillHead stops a specific encoding process
// Stream is assumed to be locked by the caller.
func (s *Stream) KillHead(encoderID int) {
	if s.heads[encoderID] == DeletedHead || s.heads[encoderID].command == nil {
		return
	}

	s.heads[encoderID].command.Process.Signal(os.Interrupt)
	s.heads[encoderID] = DeletedHead
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getKeyframes retrieves keyframes from the database, falling back to empty slice on error
func getKeyframes(wrapper *StreamWrapper) []float64 {
	assetKeyframes, err := wrapper.config.Dao.GetAssetKeyframes(context.Background(), wrapper.assetID)
	if err != nil {
		wrapper.config.Logger.Error().
			Err(err).
			Str("asset_id", wrapper.assetID).
			Str("path", wrapper.Info.Path).
			Msg("Failed to get keyframes")
		return []float64{}
	}

	if assetKeyframes != nil && len(assetKeyframes.Keyframes) > 0 {
		return assetKeyframes.Keyframes
	}

	return []float64{}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// toSegmentStr converts keyframe timestamps to a comma-separated string
func toSegmentStr(segments []float64) string {
	return strings.Join(utils.Map(segments, func(seg float64) string {
		return fmt.Sprintf("%.6f", seg)
	}), ",")
}
