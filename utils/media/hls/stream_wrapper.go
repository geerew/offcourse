package hls

import (
	"fmt"
	"strings"

	"github.com/geerew/off-course/utils/concurrency"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// StreamWrapper represents a file being transcoded into HLS streams
type StreamWrapper struct {
	config  *TranscoderConfig
	assetID string
	err     error
	Out     string
	Info    *MediaInfo
	videos  concurrency.Map[VideoKey, *VideoStream]
	audios  concurrency.Map[uint32, *AudioStream]
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MediaInfo represents media file information extracted for HLS
type MediaInfo struct {
	Path     string
	Duration float64
	Videos   []Video
	Audios   []Audio
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Video represents video metadata for a single video track
type Video struct {
	Index     uint32
	Title     *string
	Language  *string
	Codec     string
	MimeCodec *string
	Width     uint32
	Height    uint32
	Bitrate   uint32
	IsDefault bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Audio represents audio metadata for a single audio track
type Audio struct {
	Index     uint32
	Title     *string
	Language  *string
	Codec     string
	MimeCodec *string
	Bitrate   uint32
	IsDefault bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Kill stops all video and audio streams for this file
func (sw *StreamWrapper) Kill() {
	sw.videos.ForEach(func(_ VideoKey, s *VideoStream) {
		s.Kill()
	})
	sw.audios.ForEach(func(_ uint32, s *AudioStream) {
		s.Kill()
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Destroy removes all transcoded files from the cache directory
func (sw *StreamWrapper) Destroy() {
	sw.config.Logger.Debug().
		Str("asset_id", sw.assetID).
		Str("path", sw.Info.Path).
		Msg("Removing all transcode cache files")
	sw.Kill()
	_ = sw.config.FS.RemoveAll(sw.Out)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetMasterPlaylistMulti generates the HLS master playlist with multiple quality options
//
// TODO Support multiples audio qualities (and original)
func (sw *StreamWrapper) GetMasterPlaylistMulti(assetID string) string {
	master := "#EXTM3U\n"

	// Add audio media groups
	for _, audio := range sw.Info.Audios {
		master += "#EXT-X-MEDIA:TYPE=AUDIO,"
		master += "GROUP-ID=\"audio\","
		if audio.Language != nil {
			master += fmt.Sprintf("LANGUAGE=\"%s\",", *audio.Language)
		}
		if audio.Title != nil {
			master += fmt.Sprintf("NAME=\"%s\",", *audio.Title)
		} else if audio.Language != nil {
			master += fmt.Sprintf("NAME=\"%s\",", *audio.Language)
		} else {
			master += fmt.Sprintf("NAME=\"Audio %d\",", audio.Index)
		}
		if audio.IsDefault {
			master += "DEFAULT=YES,"
		}
		master += "CHANNELS=\"2\","
		master += fmt.Sprintf("URI=\"audio/%d/index.m3u8\"\n", audio.Index)
	}

	master += "\n"

	// codec is the prefix + the level, the level is not part of the codec we want to compare for the same_codec check bellow
	transcode_prefix := "avc1.6400"
	transcode_codec := transcode_prefix + "28"
	audio_codec := "mp4a.40.2"

	var def_video *Video
	for _, video := range sw.Info.Videos {
		if video.IsDefault {
			def_video = &video
			break
		}
	}
	if def_video == nil && len(sw.Info.Videos) > 0 {
		def_video = &sw.Info.Videos[0]
	}

	if def_video != nil {
		qualities := sw.GetQualities()
		aspectRatio := float32(def_video.Width) / float32(def_video.Height)

		// Generate stream variants for each quality
		for _, quality := range qualities {
			if quality == Original {
				// original quality stream
				bitrate := float64(def_video.Bitrate)
				master += "#EXT-X-STREAM-INF:"
				// For original quality, use the video's actual bitrate
				master += fmt.Sprintf("AVERAGE-BANDWIDTH=%d,", int(bitrate*0.8))
				master += fmt.Sprintf("BANDWIDTH=%d,", int(bitrate))
				master += fmt.Sprintf("RESOLUTION=%dx%d,", def_video.Width, def_video.Height)
				if def_video.MimeCodec != nil {
					master += fmt.Sprintf("CODECS=\"%s\",", strings.Join([]string{*def_video.MimeCodec, audio_codec}, ","))
				} else {
					master += fmt.Sprintf("CODECS=\"%s\",", strings.Join([]string{transcode_codec, audio_codec}, ","))
				}
				master += "AUDIO=\"audio\","
				master += "CLOSED-CAPTIONS=NONE\n"
				master += fmt.Sprintf("/api/hls/%s/video/%d/%s/index.m3u8\n", assetID, def_video.Index, quality)
				continue
			}

			// transcoded quality streams
			master += "#EXT-X-STREAM-INF:"
			master += fmt.Sprintf("AVERAGE-BANDWIDTH=%d,", quality.AverageBitrate())
			master += fmt.Sprintf("BANDWIDTH=%d,", quality.MaxBitrate())
			master += fmt.Sprintf("RESOLUTION=%dx%d,", int(aspectRatio*float32(quality.Height())+0.5), quality.Height())
			master += fmt.Sprintf("CODECS=\"%s\",", strings.Join([]string{transcode_codec, audio_codec}, ","))
			master += "AUDIO=\"audio\","
			master += "CLOSED-CAPTIONS=NONE\n"
			master += fmt.Sprintf("/api/hls/%s/video/%d/%s/index.m3u8\n", assetID, def_video.Index, quality)
		}
	}

	return master
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetMasterPlaylistSingle returns a simplified master playlist with only one stream
// - Mobile/tablet: Returns the highest available transcoded quality
// - Desktop: Returns the original quality
func (sw *StreamWrapper) GetMasterPlaylistSingle(assetID string, isMobile bool) string {
	master := "#EXTM3U\n"

	// Add audio media groups
	for _, audio := range sw.Info.Audios {
		master += "#EXT-X-MEDIA:TYPE=AUDIO,"
		master += "GROUP-ID=\"audio\","
		if audio.Language != nil {
			master += fmt.Sprintf("LANGUAGE=\"%s\",", *audio.Language)
		}
		if audio.Title != nil {
			master += fmt.Sprintf("NAME=\"%s\",", *audio.Title)
		} else if audio.Language != nil {
			master += fmt.Sprintf("NAME=\"%s\",", *audio.Language)
		} else {
			master += fmt.Sprintf("NAME=\"Audio %d\",", audio.Index)
		}
		if audio.IsDefault {
			master += "DEFAULT=YES,"
		}
		master += "CHANNELS=\"2\","
		master += fmt.Sprintf("URI=\"audio/%d/index.m3u8\"\n", audio.Index)
	}

	master += "\n"

	var def_video *Video
	for _, video := range sw.Info.Videos {
		if video.IsDefault {
			def_video = &video
			break
		}
	}
	if def_video == nil && len(sw.Info.Videos) > 0 {
		def_video = &sw.Info.Videos[0]
	}

	if def_video == nil {
		return master
	}

	audio_codec := "mp4a.40.2"
	transcode_codec := "avc1.42E01E"

	var selectedQuality Quality
	var selectedBitrate float64
	var selectedResolution string

	if isMobile {
		// For mobile, select the highest transcoded quality (not original)
		qualities := sw.GetQualities()
		selectedQuality = GetHighestTranscodedQuality(qualities)
		selectedBitrate = float64(selectedQuality.MaxBitrate())
		selectedResolution = fmt.Sprintf("%dx%d", int(float32(def_video.Width)*float32(selectedQuality.Height())/float32(def_video.Height)+0.5), selectedQuality.Height())
	} else {
		// For desktop, select original quality
		selectedQuality = Original
		selectedBitrate = float64(def_video.Bitrate)
		selectedResolution = fmt.Sprintf("%dx%d", def_video.Width, def_video.Height)
	}

	// Generate the single stream entry
	master += "#EXT-X-STREAM-INF:"
	master += fmt.Sprintf("AVERAGE-BANDWIDTH=%d,", int(selectedBitrate*0.8))
	master += fmt.Sprintf("BANDWIDTH=%d,", int(selectedBitrate))
	master += fmt.Sprintf("RESOLUTION=%s,", selectedResolution)

	if selectedQuality == Original {
		if def_video.MimeCodec != nil {
			master += fmt.Sprintf("CODECS=\"%s\",", strings.Join([]string{*def_video.MimeCodec, audio_codec}, ","))
		} else {
			master += fmt.Sprintf("CODECS=\"%s\",", strings.Join([]string{transcode_codec, audio_codec}, ","))
		}
	} else {
		master += fmt.Sprintf("CODECS=\"%s\",", strings.Join([]string{transcode_codec, audio_codec}, ","))
	}

	master += "AUDIO=\"audio\","
	master += "CLOSED-CAPTIONS=NONE\n"
	master += fmt.Sprintf("/api/hls/%s/video/%d/%s/index.m3u8\n", assetID, def_video.Index, selectedQuality)

	return master
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getVideoStream returns a video stream for the given index and quality
func (sw *StreamWrapper) getVideoStream(idx uint32, quality Quality) (*VideoStream, error) {
	stream, _ := sw.videos.GetOrSetFn(VideoKey{idx, quality}, func() *VideoStream {
		ret, _ := NewVideoStream(sw, idx, quality)
		return ret
	})

	return stream, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetVideoIndex returns the video index playlist for a given variant
func (sw *StreamWrapper) GetVideoIndex(idx uint32, quality Quality) (string, error) {
	stream, err := sw.getVideoStream(idx, quality)
	if err != nil {
		return "", err
	}

	return stream.GetIndex()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetVideoSegment returns a video segment path, transcoding if necessary
func (sw *StreamWrapper) GetVideoSegment(idx uint32, quality Quality, segment int32) (string, error) {
	stream, err := sw.getVideoStream(idx, quality)
	if err != nil {
		return "", err
	}

	return stream.GetSegment(segment)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getAudioStream returns an audio stream for the given audio index
func (sw *StreamWrapper) getAudioStream(audio uint32) (*AudioStream, error) {
	stream, _ := sw.audios.GetOrSetFn(audio, func() *AudioStream {
		ret, _ := NewAudioStream(sw, audio)
		return ret
	})

	return stream, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAudioIndex returns the audio index playlist for the given audio index
func (sw *StreamWrapper) GetAudioIndex(audio uint32) (string, error) {
	stream, err := sw.getAudioStream(audio)
	if err != nil {
		return "", err
	}

	return stream.GetIndex()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAudioSegment returns an audio segment path, transcoding if necessary
func (sw *StreamWrapper) GetAudioSegment(audio uint32, segment int32) (string, error) {
	stream, err := sw.getAudioStream(audio)
	if err != nil {
		return "", err
	}

	return stream.GetSegment(segment)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetQualities returns the available qualities for the video
func (sw *StreamWrapper) GetQualities() []Quality {
	var def_video *Video
	for _, video := range sw.Info.Videos {
		if video.IsDefault {
			def_video = &video
			break
		}
	}
	if def_video == nil && len(sw.Info.Videos) > 0 {
		def_video = &sw.Info.Videos[0]
	}

	if def_video == nil {
		return []Quality{}
	}

	var qualities []Quality

	qualities = append(qualities, Original)

	// Add qualities from highest to lowest
	for i := len(Qualities) - 1; i >= 0; i-- {
		q := Qualities[i]
		if q.Height() <= def_video.Height {
			qualities = append(qualities, q)
		}
	}

	return qualities
}
