package dao

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateAssetMetadata inserts a new asset metadata record
func (dao *DAO) CreateAssetMetadata(ctx context.Context, metadata *models.AssetMetadata) error {
	if metadata == nil {
		return utils.ErrNilPtr
	}

	if metadata.VideoMetadata == nil && metadata.AudioMetadata == nil {
		return nil
	}

	return dao.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Create video metadata (if present)
		if metadata.VideoMetadata != nil {
			videoMetadata := metadata.VideoMetadata

			if videoMetadata.ID == "" {
				videoMetadata.RefreshId()
			}

			videoMetadata.RefreshCreatedAt()
			videoMetadata.RefreshUpdatedAt()

			builderOpts := newBuilderOptions(models.ASSET_METADATA_VIDEO_TABLE).
				WithData(
					map[string]interface{}{
						models.BASE_ID:                          videoMetadata.ID,
						models.ASSET_METADATA_VIDEO_ASSET_ID:    metadata.AssetID,
						models.ASSET_METADATA_VIDEO_DURATION:    videoMetadata.DurationSec,
						models.ASSET_METADATA_VIDEO_CONTAINER:   videoMetadata.Container,
						models.ASSET_METADATA_VIDEO_MIME_TYPE:   videoMetadata.MIMEType,
						models.ASSET_METADATA_VIDEO_SIZE_BYTES:  videoMetadata.SizeBytes,
						models.ASSET_METADATA_VIDEO_OVERALL_BPS: videoMetadata.OverallBPS,
						models.ASSET_METADATA_VIDEO_CODEC:       videoMetadata.VideoCodec,
						models.ASSET_METADATA_VIDEO_WIDTH:       videoMetadata.Width,
						models.ASSET_METADATA_VIDEO_HEIGHT:      videoMetadata.Height,
						models.ASSET_METADATA_VIDEO_FPS_NUM:     videoMetadata.FPSNum,
						models.ASSET_METADATA_VIDEO_FPS_DEN:     videoMetadata.FPSDen,
						models.BASE_CREATED_AT:                  videoMetadata.CreatedAt,
						models.BASE_UPDATED_AT:                  videoMetadata.UpdatedAt,
					})

			err := createGeneric(txCtx, dao, *builderOpts)
			if err != nil {
				return err
			}
		}

		// Create audio metadata (if present)
		if metadata.AudioMetadata != nil {
			audioMetadata := metadata.AudioMetadata

			if audioMetadata.ID == "" {
				audioMetadata.RefreshId()
			}

			audioMetadata.RefreshCreatedAt()
			audioMetadata.RefreshUpdatedAt()

			builderOpts := newBuilderOptions(models.ASSET_METADATA_AUDIO_TABLE).
				WithData(
					map[string]interface{}{
						models.BASE_ID:                             audioMetadata.ID,
						models.ASSET_METADATA_AUDIO_ASSET_ID:       metadata.AssetID,
						models.ASSET_METADATA_AUDIO_LANGUAGE:       audioMetadata.Language,
						models.ASSET_METADATA_AUDIO_CODEC:          audioMetadata.Codec,
						models.ASSET_METADATA_AUDIO_PROFILE:        audioMetadata.Profile,
						models.ASSET_METADATA_AUDIO_CHANNELS:       audioMetadata.Channels,
						models.ASSET_METADATA_AUDIO_CHANNEL_LAYOUT: audioMetadata.ChannelLayout,
						models.ASSET_METADATA_AUDIO_SAMPLE_RATE:    audioMetadata.SampleRate,
						models.ASSET_METADATA_AUDIO_BIT_RATE:       audioMetadata.BitRate,
						models.BASE_CREATED_AT:                     audioMetadata.CreatedAt,
						models.BASE_UPDATED_AT:                     audioMetadata.UpdatedAt,
					})

			err := createGeneric(txCtx, dao, *builderOpts)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAssetMetadata returns a single asset metadata record
func (dao *DAO) GetAssetMetadata(ctx context.Context, assetID string) (*models.AssetMetadata, error) {
	if assetID == "" {
		return nil, utils.ErrId
	}

	videoMap, err := listVideoMetadataByAssetIDs(ctx, dao, []string{assetID})
	if err != nil {
		return nil, err
	}

	audioMap, err := listAudioMetadataByAssetIDs(ctx, dao, []string{assetID})
	if err != nil {
		return nil, err
	}

	videoMetadata := videoMap[assetID]
	audioMetadata := audioMap[assetID]
	if videoMetadata == nil && audioMetadata == nil {
		return nil, nil
	}

	out := &models.AssetMetadata{AssetID: assetID}
	if videoMetadata != nil {
		out.VideoMetadata = videoMetadata
	}

	if audioMetadata != nil {
		audioMetadata.FixChannelsFromLayout()
		out.AudioMetadata = audioMetadata
	}

	return out, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListAssetMetadataByAssetIDs returns asset metadata records for the given IDs
func (dao *DAO) ListAssetMetadataByAssetIDs(ctx context.Context, assetIDs []string) (map[string]*models.AssetMetadata, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	videoMap, err := listVideoMetadataByAssetIDs(ctx, dao, assetIDs)
	if err != nil {
		return nil, err
	}

	audioMap, err := listAudioMetadataByAssetIDs(ctx, dao, assetIDs)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*models.AssetMetadata, len(assetIDs))
	for _, id := range assetIDs {
		videoMetadata := videoMap[id]
		audioMetadata := audioMap[id]
		if videoMetadata == nil && audioMetadata == nil {
			continue
		}

		m := &models.AssetMetadata{AssetID: id}
		if videoMetadata != nil {
			m.VideoMetadata = videoMetadata
		}

		if audioMetadata != nil {
			audioMetadata.FixChannelsFromLayout()
			m.AudioMetadata = audioMetadata
		}

		out[id] = m
	}

	return out, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// listVideoMetadataByAssetIDs lists the video metadata for the given asset IDs
func listVideoMetadataByAssetIDs(ctx context.Context, dao *DAO, assetIDs []string) (map[string]*models.VideoMetadata, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_METADATA_VIDEO_TABLE_ASSET_ID: assetIDs})
	builderOpts := newBuilderOptions(models.ASSET_METADATA_VIDEO_TABLE).
		WithColumns(models.VideoMetadataColumns()...).
		SetDbOpts(dbOpts)

	rows, err := listGeneric[models.VideoMetadata](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*models.VideoMetadata, len(rows))
	for _, r := range rows {
		if r != nil {
			out[r.AssetID] = r
		}
	}

	return out, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// listAudioMetadataByAssetIDs lists the audio metadata for the given asset IDs
func listAudioMetadataByAssetIDs(ctx context.Context, dao *DAO, assetIDs []string) (map[string]*models.AudioMetadata, error) {
	if len(assetIDs) == 0 {
		return nil, nil
	}

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_METADATA_AUDIO_TABLE_ASSET_ID: assetIDs})
	builderOpts := newBuilderOptions(models.ASSET_METADATA_AUDIO_TABLE).
		WithColumns(models.AudioMetadataColumns()...).
		SetDbOpts(dbOpts)

	rows, err := listGeneric[models.AudioMetadata](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*models.AudioMetadata, len(rows))
	for _, r := range rows {
		if r != nil {
			out[r.AssetID] = r
		}
	}

	return out, nil
}
