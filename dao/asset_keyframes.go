package dao

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateAssetKeyframes inserts a new asset keyframes record
func (dao *DAO) CreateAssetKeyframes(ctx context.Context, keyframes *models.AssetKeyframes) error {
	if keyframes == nil {
		return utils.ErrNilPtr
	}

	if keyframes.AssetID == "" {
		return utils.ErrAssetId
	}

	if err := keyframes.Keyframes.Validate(); err != nil {
		return fmt.Errorf("invalid keyframes: %w", err)
	}

	if keyframes.ID == "" {
		keyframes.RefreshId()
	}

	keyframes.RefreshCreatedAt()
	keyframes.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.ASSET_KEYFRAMES_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:                     keyframes.ID,
				models.ASSET_KEYFRAMES_ASSET_ID:    keyframes.AssetID,
				models.ASSET_KEYFRAMES_KEYFRAMES:   keyframes.Keyframes,
				models.ASSET_KEYFRAMES_IS_COMPLETE: keyframes.IsComplete,
				models.BASE_CREATED_AT:             keyframes.CreatedAt,
				models.BASE_UPDATED_AT:             keyframes.UpdatedAt,
			})

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAssetKeyframes retrieves asset keyframes by asset ID
func (dao *DAO) GetAssetKeyframes(ctx context.Context, assetID string) (*models.AssetKeyframes, error) {
	if assetID == "" {
		return nil, utils.ErrAssetId
	}

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_KEYFRAMES_ASSET_ID: assetID})

	builderOpts := newBuilderOptions(models.ASSET_KEYFRAMES_TABLE).
		WithColumns(models.AssetKeyframesColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	keyframes, err := getGeneric[models.AssetKeyframes](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	return keyframes, nil
}
