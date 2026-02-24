package dao

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateAsset inserts a new asset record
func (dao *DAO) CreateAsset(ctx context.Context, asset *models.Asset) error {

	if err := assetValidation(asset); err != nil {
		return err
	}

	if asset.ID == "" {
		asset.RefreshId()
	}

	asset.RefreshCreatedAt()
	asset.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.ASSET_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:          asset.ID,
				models.ASSET_COURSE_ID:  asset.CourseID,
				models.ASSET_LESSON_ID:  asset.LessonID,
				models.ASSET_TITLE:      asset.Title,
				models.ASSET_PREFIX:     asset.Prefix,
				models.ASSET_SUB_PREFIX: asset.SubPrefix,
				models.ASSET_SUB_TITLE:  asset.SubTitle,
				models.ASSET_MODULE:     asset.Module,
				models.ASSET_TYPE:       asset.Type,
				models.ASSET_PATH:       asset.Path,
				models.ASSET_FILE_SIZE:  asset.FileSize,
				models.ASSET_MOD_TIME:   asset.ModTime,
				models.ASSET_HASH:       asset.Hash,
				models.BASE_CREATED_AT:  asset.CreatedAt,
				models.BASE_UPDATED_AT:  asset.UpdatedAt,
			},
		)

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CountAssets counts the number of asset records
func (dao *DAO) CountAssets(ctx context.Context, dbOpts *Options) (int, error) {
	builderOpts := newBuilderOptions(models.ASSET_TABLE).SetDbOpts(dbOpts)
	return countGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAsset gets a record from the assets table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
//
// Asset progress is not included by default. It can be enabled by calling `WithUserProgress()` on the options
// Asset metadata is not included by default. It can be enabled by calling `WithAssetMetadata()` on the options
func (dao *DAO) GetAsset(ctx context.Context, dbOpts *Options) (*models.Asset, error) {
	builderOpts := newBuilderOptions(models.ASSET_TABLE).
		WithColumns(models.AssetColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress
	includeMetadata := dbOpts != nil && dbOpts.IncludeAssetMetadata

	// When no relations are included, use a simpler query
	if !includeProgress && !includeMetadata {
		return getGeneric[models.Asset](ctx, dao, *builderOpts)
	}

	// When progress is requested, validate the principal before fetching
	// the asset
	var principal types.Principal
	if includeProgress {
		var err error
		principal, err = principalFromCtx(ctx)
		if err != nil {
			return nil, err
		}
	}

	asset, err := getGeneric[models.Asset](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if asset == nil {
		return nil, nil
	}

	if includeProgress {
		dbOpts := NewOptions().WithWhere(squirrel.And{
			squirrel.Eq{models.ASSET_PROGRESS_ASSET_ID: asset.ID},
			squirrel.Eq{models.ASSET_PROGRESS_USER_ID: principal.UserID},
		})

		assetProgress, err := dao.GetAssetProgress(ctx, dbOpts)
		if err != nil {
			return nil, err
		}

		if assetProgress != nil {
			asset.Progress = assetProgress
		} else {
			asset.Progress = &models.AssetProgress{AssetID: asset.ID, UserID: principal.UserID}
		}
	}

	if includeMetadata {
		assetMetadata, err := dao.GetAssetMetadata(ctx, asset.ID)
		if err != nil {
			return nil, err
		}

		asset.AssetMetadata = assetMetadata
	}

	return asset, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListAssets gets all records from the assets table based upon the where clause and pagination
// in the options
//
// Asset progress is not included by default. It can be enabled by calling `WithUserProgress()` on the options
// Asset metadata is not included by default. It can be enabled by calling `WithAssetMetadata()` on the options
func (dao *DAO) ListAssets(ctx context.Context, dbOpts *Options) ([]*models.Asset, error) {
	builderOpts := newBuilderOptions(models.ASSET_TABLE).
		WithColumns(models.AssetColumns()...).
		SetDbOpts(dbOpts)

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress
	includeMetadata := dbOpts != nil && dbOpts.IncludeAssetMetadata

	// Validate principal early when progress is requested
	var principal types.Principal
	if includeProgress {
		var err error
		principal, err = principalFromCtx(ctx)
		if err != nil {
			return nil, err
		}
	}

	// When no relations are included, use a simpler query
	if !includeProgress && !includeMetadata {
		return listGeneric[models.Asset](ctx, dao, *builderOpts)
	}

	records, err := listGeneric[models.Asset](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, nil
	}

	if includeProgress {
		assetIDs := make([]string, len(records))
		for i, a := range records {
			assetIDs[i] = a.ID
		}

		dbOpts := NewOptions().WithWhere(squirrel.And{
			squirrel.Eq{models.ASSET_PROGRESS_ASSET_ID: assetIDs},
			squirrel.Eq{models.ASSET_PROGRESS_USER_ID: principal.UserID},
		})

		progressRows, err := dao.ListAssetProgress(ctx, dbOpts)
		if err != nil {
			return nil, err
		}

		progressMap := make(map[string]*models.AssetProgress, len(assetIDs))
		for _, id := range assetIDs {
			progressMap[id] = &models.AssetProgress{AssetID: id, UserID: principal.UserID}
		}

		for _, p := range progressRows {
			if p != nil {
				progressMap[p.AssetID] = p
			}
		}

		for _, a := range records {
			a.Progress = progressMap[a.ID]
		}
	}

	if includeMetadata {
		assetIDs := make([]string, len(records))

		for i, a := range records {
			assetIDs[i] = a.ID
		}

		metaMap, err := dao.GetAssetMetadataByAssetIDs(ctx, assetIDs)
		if err != nil {
			return nil, err
		}

		for _, a := range records {
			a.AssetMetadata = metaMap[a.ID]
		}
	}

	return records, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UpdateAsset updates an asset record
func (dao *DAO) UpdateAsset(ctx context.Context, asset *models.Asset) error {
	if err := assetValidation(asset); err != nil {
		return err
	}

	if asset.ID == "" {
		return utils.ErrId
	}

	asset.RefreshUpdatedAt()

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: asset.ID})

	builderOpts := newBuilderOptions(models.ASSET_TABLE).
		WithData(
			map[string]interface{}{
				models.ASSET_LESSON_ID:  asset.LessonID,
				models.ASSET_TITLE:      asset.Title,
				models.ASSET_PREFIX:     asset.Prefix,
				models.ASSET_SUB_PREFIX: asset.SubPrefix,
				models.ASSET_SUB_TITLE:  asset.SubTitle,
				models.ASSET_MODULE:     asset.Module,
				models.ASSET_TYPE:       asset.Type,
				models.ASSET_PATH:       asset.Path,
				models.ASSET_FILE_SIZE:  asset.FileSize,
				models.ASSET_MOD_TIME:   asset.ModTime,
				models.ASSET_HASH:       asset.Hash,
				models.BASE_UPDATED_AT:  asset.UpdatedAt,
			},
		).
		SetDbOpts(dbOpts)

	_, err := updateGeneric(ctx, dao, *builderOpts)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteAssets deletes records from the assets table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteAssets(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.ASSET_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// assetValidation validates the asset fields
func assetValidation(asset *models.Asset) error {
	if asset == nil {
		return utils.ErrNilPtr
	}

	if asset.CourseID == "" {
		return utils.ErrCourseId
	}

	if asset.LessonID == "" {
		return utils.ErrLessonId
	}

	if asset.Title == "" {
		return utils.ErrTitle
	}

	if !asset.Prefix.Valid || asset.Prefix.Int16 < 0 {
		return utils.ErrPrefix
	}

	if asset.Path == "" {
		return utils.ErrPath
	}

	return nil
}
