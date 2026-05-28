package hls

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/filesystem"
	"github.com/geerew/off-course/utils/media"
	"github.com/geerew/off-course/utils/concurrency"
	"github.com/geerew/off-course/utils/logger"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Transcoder manages all HLS transcoding operations for a given asset
type Transcoder struct {
	config    *TranscoderConfig
	cachePath string
	streams   concurrency.Map[string, *StreamWrapper]
	assetChan chan string
	tracker   *Tracker
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TranscoderConfig defines the configuration for a Transcoder
type TranscoderConfig struct {
	CachePath string
	HwAccel   HwAccelT
	FS        *filesystem.FS
	Logger    *logger.Logger
	Dao       *dao.DAO
	Tools     *media.Tools
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewTranscoder creates a new Transcoder and prepares the cache directory
func NewTranscoder(config *TranscoderConfig) (*Transcoder, error) {
	// Use relative paths for in-memory filesystems
	var cachePath string
	if _, ok := config.FS.Fs.(*afero.MemMapFs); ok {
		// In-memory filesystem
		cachePath = filepath.Join(config.CachePath, "hls")
	} else {
		// Real filesystem
		absDataDir, err := filepath.Abs(config.CachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache path: %w", err)
		}

		cachePath = filepath.Join(absDataDir, "hls")
	}

	config.FS.MkdirAll(cachePath, 0o755)

	// Empty the cache directory
	err := config.FS.RemoveAllContents(cachePath)
	if err != nil {
		return nil, err
	}

	transcoder := &Transcoder{
		config:    config,
		cachePath: cachePath,
		streams:   concurrency.NewMap[string, *StreamWrapper](),
		assetChan: make(chan string, 10),
	}

	// Start tracker
	transcoder.tracker = NewTracker(transcoder)

	return transcoder, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// newStreamWrapper creates a new StreamWrapper and fetches metadata from the database
func (t *Transcoder) newStreamWrapper(ctx context.Context, path string, assetID string) *StreamWrapper {
	streamWrapper := &StreamWrapper{
		config:  t.config,
		Out:     filepath.Join(t.cachePath, assetID),
		videos:  concurrency.NewMap[VideoKey, *VideoStream](),
		audios:  concurrency.NewMap[uint32, *AudioStream](),
		assetID: assetID,
	}

	// Get asset with metadata from database
	asset, err := t.config.Dao.GetAsset(ctx, dao.NewOptions().
		WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: assetID}).
		WithAssetMetadata())
	if err != nil {
		t.config.Logger.Error().
			Err(err).
			Str("asset_id", assetID).
			Str("path", path).
			Msg("Failed to get asset metadata")
		streamWrapper.err = err
		return streamWrapper
	}

	// Convert database models to HLS models
	var videos []Video
	var audios []Audio

	// Video metadata
	if asset.AssetMetadata != nil && asset.AssetMetadata.VideoMetadata != nil {
		videoMeta := asset.AssetMetadata.VideoMetadata
		video := Video{
			Index:     0,
			Title:     nil,
			Language:  nil,
			Codec:     videoMeta.VideoCodec,
			MimeCodec: nil,
			Width:     uint32(videoMeta.Width),
			Height:    uint32(videoMeta.Height),
			Bitrate:   uint32(videoMeta.OverallBPS),
			IsDefault: true,
		}
		videos = append(videos, video)
	}

	// Audio metadata
	if asset.AssetMetadata != nil && asset.AssetMetadata.AudioMetadata != nil {
		audioMeta := asset.AssetMetadata.AudioMetadata
		audio := Audio{
			Index:     0,
			Title:     nil,
			Language:  nil,
			Codec:     audioMeta.Codec,
			MimeCodec: nil,
			Bitrate:   uint32(audioMeta.BitRate),
			IsDefault: true,
		}
		audios = append(audios, audio)
	}

	// Duration
	duration := 0.0
	if asset.AssetMetadata != nil && asset.AssetMetadata.VideoMetadata != nil {
		duration = float64(asset.AssetMetadata.VideoMetadata.DurationSec)
	}

	info := &MediaInfo{
		Path:     path,
		Duration: duration,
		Videos:   videos,
		Audios:   audios,
	}
	streamWrapper.Info = info

	return streamWrapper
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getStreamWrapper returns an existing StreamWrapper for the asset or creates one
//
// It blocks until the StreamWrapper is ready or returns an error
func (t *Transcoder) getStreamWrapper(ctx context.Context, path string, assetID string) (*StreamWrapper, error) {
	streamWrapper, _ := t.streams.GetOrSet(assetID, t.newStreamWrapper(ctx, path, assetID))

	if streamWrapper.err != nil {
		t.streams.Remove(assetID)
		return nil, streamWrapper.err
	}

	return streamWrapper, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetMasterPlaylistMulti returns the master HLS playlist with multiple quality options
func (t *Transcoder) GetMasterPlaylistMulti(ctx context.Context, path string, assetID string) (string, error) {
	streamWrapper, err := t.getStreamWrapper(ctx, path, assetID)
	if err != nil {
		return "", err
	}

	t.assetChan <- assetID
	return streamWrapper.GetMasterPlaylistMulti(assetID), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetMasterPlaylistSingle returns a master playlist with only one stream
func (t *Transcoder) GetMasterPlaylistSingle(ctx context.Context, path string, assetID string, isMobile bool) (string, error) {
	streamWrapper, err := t.getStreamWrapper(ctx, path, assetID)
	if err != nil {
		return "", err
	}

	t.assetChan <- assetID
	return streamWrapper.GetMasterPlaylistSingle(assetID, isMobile), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetVideoIndex returns the video variant index playlist for a specific quality
func (t *Transcoder) GetVideoIndex(
	ctx context.Context,
	path string,
	video uint32,
	quality Quality,
	assetID string,
) (string, error) {
	streamWrapper, err := t.getStreamWrapper(ctx, path, assetID)
	if err != nil {
		return "", err
	}

	t.assetChan <- assetID
	return streamWrapper.GetVideoIndex(video, quality)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetVideoSegment returns the path to a requested video segment, transcoding if necessary
func (t *Transcoder) GetVideoSegment(
	ctx context.Context,
	path string,
	video uint32,
	quality Quality,
	segment int32,
	assetID string,
) (string, error) {
	streamWrapper, err := t.getStreamWrapper(ctx, path, assetID)
	if err != nil {
		return "", err
	}

	t.assetChan <- assetID
	return streamWrapper.GetVideoSegment(video, quality, segment)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAudioIndex returns the audio index playlist for the specified audio index
func (t *Transcoder) GetAudioIndex(
	ctx context.Context,
	path string,
	audio uint32,
	assetID string,
) (string, error) {
	streamWrapper, err := t.getStreamWrapper(ctx, path, assetID)
	if err != nil {
		return "", err
	}

	t.assetChan <- assetID
	return streamWrapper.GetAudioIndex(audio)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAudioSegment returns the path to a requested audio segment, transcoding if necessary
func (t *Transcoder) GetAudioSegment(
	ctx context.Context,
	path string,
	audio uint32,
	segment int32,
	assetID string,
) (string, error) {
	streamWrapper, err := t.getStreamWrapper(ctx, path, assetID)
	if err != nil {
		return "", err
	}

	t.assetChan <- assetID
	return streamWrapper.GetAudioSegment(audio, segment)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetQualities returns the available qualities for a video
func (t *Transcoder) GetQualities(ctx context.Context, path string, assetID string) ([]Quality, error) {
	streamWrapper, err := t.getStreamWrapper(ctx, path, assetID)
	if err != nil {
		return nil, err
	}

	return streamWrapper.GetQualities(), nil
}
