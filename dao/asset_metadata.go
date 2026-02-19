package dao

import (
	"context"
	"fmt"

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

	// Nothing to do
	if metadata.VideoMetadata == nil && metadata.AudioMetadata == nil {
		return nil
	}

	return dao.db.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Create video metadata (if present)
		if metadata.VideoMetadata != nil {
			vm := metadata.VideoMetadata

			if vm.ID == "" {
				vm.RefreshId()
			}

			vm.RefreshCreatedAt()
			vm.RefreshUpdatedAt()

			builderOpts := newBuilderOptions(models.MEDIA_VIDEO_TABLE).
				WithData(
					map[string]interface{}{
						models.BASE_ID:                 vm.ID,
						models.META_ASSET_ID:           metadata.AssetID,
						models.MEDIA_VIDEO_DURATION:    vm.DurationSec,
						models.MEDIA_VIDEO_CONTAINER:   vm.Container,
						models.MEDIA_VIDEO_MIME_TYPE:   vm.MIMEType,
						models.MEDIA_VIDEO_SIZE_BYTES:  vm.SizeBytes,
						models.MEDIA_VIDEO_OVERALL_BPS: vm.OverallBPS,
						models.MEDIA_VIDEO_CODEC:       vm.VideoCodec,
						models.MEDIA_VIDEO_WIDTH:       vm.Width,
						models.MEDIA_VIDEO_HEIGHT:      vm.Height,
						models.MEDIA_VIDEO_FPS_NUM:     vm.FPSNum,
						models.MEDIA_VIDEO_FPS_DEN:     vm.FPSDen,
						models.BASE_CREATED_AT:         vm.CreatedAt,
						models.BASE_UPDATED_AT:         vm.UpdatedAt,
					})

			err := createGeneric(txCtx, dao, *builderOpts)
			if err != nil {
				return err
			}
		}

		// Create audio metadata (if present)
		if metadata.AudioMetadata != nil {
			am := metadata.AudioMetadata

			if am.ID == "" {
				am.RefreshId()
			}

			am.RefreshCreatedAt()
			am.RefreshUpdatedAt()

			builderOpts := newBuilderOptions(models.MEDIA_AUDIO_TABLE).
				WithData(
					map[string]interface{}{
						models.BASE_ID:                    am.ID,
						models.META_ASSET_ID:              metadata.AssetID,
						models.MEDIA_AUDIO_LANGUAGE:       am.Language,
						models.MEDIA_AUDIO_CODEC:          am.Codec,
						models.MEDIA_AUDIO_PROFILE:        am.Profile,
						models.MEDIA_AUDIO_CHANNELS:       am.Channels,
						models.MEDIA_AUDIO_CHANNEL_LAYOUT: am.ChannelLayout,
						models.MEDIA_AUDIO_SAMPLE_RATE:    am.SampleRate,
						models.MEDIA_AUDIO_BIT_RATE:       am.BitRate,
						models.BASE_CREATED_AT:            am.CreatedAt,
						models.BASE_UPDATED_AT:            am.UpdatedAt,
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

// GetAssetMetadata returns a single aggregate metadata record for an asset
func (dao *DAO) GetAssetMetadata(ctx context.Context, assetID string) (*models.AssetMetadata, error) {
	if assetID == "" {
		return nil, utils.ErrId
	}

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: assetID})

	builderOpts := newBuilderOptions(models.ASSET_TABLE).
		WithColumns(models.AssetMetadataRowColumns()...).
		WithLeftJoin(models.MEDIA_VIDEO_TABLE, fmt.Sprintf("%s = %s", models.ASSET_TABLE_ID, models.MEDIA_VIDEO_TABLE_ASSET_ID)).
		WithLeftJoin(models.MEDIA_AUDIO_TABLE, fmt.Sprintf("%s = %s", models.ASSET_TABLE_ID, models.MEDIA_AUDIO_TABLE_ASSET_ID)).
		SetDbOpts(dbOpts).
		WithLimit(1)

	row, err := getGeneric[models.AssetMetadataRow](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil // asset not found
	}

	// If neither joined, treat as “no metadata”
	if !row.VideoID.Valid && !row.AudioID.Valid {
		return nil, nil
	}

	return row.ToDomain(), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListAssetMetadata gets all records asset metadata based upon the where clause and pagination
// in the options
func (dao *DAO) ListAssetMetadata(ctx context.Context, dbOpts *Options) ([]*models.AssetMetadata, error) {
	builderOpts := newBuilderOptions(models.ASSET_TABLE).
		WithColumns(models.AssetMetadataRowColumns()...).
		WithLeftJoin(models.MEDIA_VIDEO_TABLE, fmt.Sprintf("%s = %s", models.ASSET_TABLE_ID, models.MEDIA_VIDEO_TABLE_ASSET_ID)).
		WithLeftJoin(models.MEDIA_AUDIO_TABLE, fmt.Sprintf("%s = %s", models.ASSET_TABLE_ID, models.MEDIA_AUDIO_TABLE_ASSET_ID)).
		SetDbOpts(dbOpts)

	rows, err := listGeneric[models.AssetMetadataRow](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	records := make([]*models.AssetMetadata, 0, len(rows))
	for i := range rows {
		r := rows[i]

		if !r.VideoID.Valid && !r.AudioID.Valid {
			continue
		}
		records = append(records, r.ToDomain())
	}

	if len(records) == 0 {
		return nil, nil
	}

	return records, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UpdateAssetMetadata updates video/audio metadata record for an asset
// - If metadata.VideoMetadata == nil, video row is untouched
// - If metadata.AudioMetadata == nil, audio row is untouched
func (dao *DAO) UpdateAssetMetadata(ctx context.Context, metadata *models.AssetMetadata) error {
	if metadata == nil {
		return utils.ErrNilPtr
	}

	if metadata.AssetID == "" {
		return utils.ErrId
	}

	// Nothing to do
	if metadata.VideoMetadata == nil && metadata.AudioMetadata == nil {
		return nil
	}

	return dao.db.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Video
		if vm := metadata.VideoMetadata; vm != nil {
			// bump updated_at for this subrecord
			vm.RefreshUpdatedAt()

			dbOpts := NewOptions().
				WithWhere(squirrel.Eq{models.MEDIA_VIDEO_TABLE_ASSET_ID: metadata.AssetID})

			builder := newBuilderOptions(models.MEDIA_VIDEO_TABLE).
				WithData(map[string]interface{}{
					models.MEDIA_VIDEO_DURATION:    vm.DurationSec,
					models.MEDIA_VIDEO_CONTAINER:   vm.Container,
					models.MEDIA_VIDEO_MIME_TYPE:   vm.MIMEType,
					models.MEDIA_VIDEO_SIZE_BYTES:  vm.SizeBytes,
					models.MEDIA_VIDEO_OVERALL_BPS: vm.OverallBPS,
					models.MEDIA_VIDEO_CODEC:       vm.VideoCodec,
					models.MEDIA_VIDEO_WIDTH:       vm.Width,
					models.MEDIA_VIDEO_HEIGHT:      vm.Height,
					models.MEDIA_VIDEO_FPS_NUM:     vm.FPSNum,
					models.MEDIA_VIDEO_FPS_DEN:     vm.FPSDen,
					models.BASE_UPDATED_AT:         vm.UpdatedAt,
				}).
				SetDbOpts(dbOpts)

			if _, err := updateGeneric(txCtx, dao, *builder); err != nil {
				return err
			}
		}

		// Audio
		if am := metadata.AudioMetadata; am != nil {
			am.RefreshUpdatedAt()

			dbOpts := NewOptions().
				WithWhere(squirrel.Eq{models.MEDIA_AUDIO_TABLE_ASSET_ID: metadata.AssetID})

			builder := newBuilderOptions(models.MEDIA_AUDIO_TABLE).
				WithData(map[string]interface{}{
					models.MEDIA_AUDIO_LANGUAGE:       am.Language,
					models.MEDIA_AUDIO_CODEC:          am.Codec,
					models.MEDIA_AUDIO_PROFILE:        am.Profile,
					models.MEDIA_AUDIO_CHANNELS:       am.Channels,
					models.MEDIA_AUDIO_CHANNEL_LAYOUT: am.ChannelLayout,
					models.MEDIA_AUDIO_SAMPLE_RATE:    am.SampleRate,
					models.MEDIA_AUDIO_BIT_RATE:       am.BitRate,
					models.BASE_UPDATED_AT:            am.UpdatedAt,
				}).
				SetDbOpts(dbOpts)

			if _, err := updateGeneric(txCtx, dao, *builder); err != nil {
				return err
			}
		}

		return nil
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteAssetMetadataByAssetIDs deletes records from the video and audio metadata tables
// for the given asset IDs
func (dao *DAO) DeleteAssetMetadataByAssetIDs(ctx context.Context, assetIDs ...string) error {
	ids := sanitizeIDs(assetIDs)
	if len(ids) == 0 {
		return nil
	}

	return dao.db.RunInTransaction(ctx, func(txCtx context.Context) error {
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.META_ASSET_ID: ids})

		// Audio metadata
		builder := newBuilderOptions(models.MEDIA_AUDIO_TABLE).SetDbOpts(dbOpts)
		sqlStr, args, _ := deleteBuilder(*builder)
		if _, err := dao.db.ExecContext(txCtx, sqlStr, args...); err != nil {
			return err
		}

		// Video metadata
		builder = newBuilderOptions(models.MEDIA_VIDEO_TABLE).SetDbOpts(dbOpts)
		sqlStr, args, _ = deleteBuilder(*builder)
		if _, err := dao.db.ExecContext(txCtx, sqlStr, args...); err != nil {
			return err
		}

		return nil
	})
}
