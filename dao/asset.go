package dao

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
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
// By default, progress is not included. Use `WithUserProgress()` on the options to include it
// By default, video metadata is not included. Use `WithAssetVideoMetadata()` on the options to include it
func (dao *DAO) GetAsset(ctx context.Context, dbOpts *Options) (*models.Asset, error) {
	builderOpts := newBuilderOptions(models.ASSET_TABLE).
		WithColumns(models.AssetColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress
	includeMetadata := dbOpts != nil && dbOpts.IncludeAssetMetadata
	includeCourse := dbOpts != nil && dbOpts.IncludeCourse
	includeLesson := dbOpts != nil && dbOpts.IncludeLesson

	// When no relations are included, use a simpler query
	if !includeProgress && !includeMetadata && !includeCourse && !includeLesson {
		return getGeneric[models.Asset](ctx, dao, *builderOpts)
	}

	// Add the progress columns and join
	if includeProgress {
		principal, err := principalFromCtx(ctx)
		if err != nil {
			return nil, err
		}

		builderOpts = builderOpts.
			WithColumns(models.AssetProgressRowColumns()...).
			WithLeftJoin(
				models.ASSET_PROGRESS_TABLE,
				fmt.Sprintf(
					"%s = %s AND %s = '%s'",
					models.ASSET_PROGRESS_TABLE_ASSET_ID,
					models.ASSET_TABLE_ID,
					models.ASSET_PROGRESS_TABLE_USER_ID,
					principal.UserID,
				),
			)
	}
	// Metadata is fetched separately via GetAssetMetadata when includeMetadata is true

	// Add lesson and course joins if enabled
	if dbOpts != nil {
		// If both course and lesson are requested, join lesson first, then course through lesson
		if dbOpts.IncludeCourse && dbOpts.IncludeLesson {
			builderOpts = builderOpts.
				WithJoin(models.LESSON_TABLE, models.ASSET_TABLE_LESSON_ID+" = "+models.LESSON_TABLE_ID).
				WithJoin(models.COURSE_TABLE, models.LESSON_TABLE_COURSE_ID+" = "+models.COURSE_TABLE_ID)
		} else if dbOpts.IncludeLesson {
			// Only lesson join
			builderOpts = builderOpts.WithJoin(models.LESSON_TABLE, models.ASSET_TABLE_LESSON_ID+" = "+models.LESSON_TABLE_ID)
		} else if dbOpts.IncludeCourse {
			// Only course join
			builderOpts = builderOpts.WithJoin(models.COURSE_TABLE, models.ASSET_TABLE_COURSE_ID+" = "+models.COURSE_TABLE_ID)
		}
	}

	row, err := getGeneric[models.AssetRow](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	a := row.ToDomain(includeProgress, includeMetadata)
	if includeMetadata && a != nil {
		meta, err := dao.GetAssetMetadata(ctx, a.ID)
		if err != nil {
			return nil, err
		}
		a.AssetMetadata = meta
	}
	return a, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListAssets gets all records from the assets table based upon the where clause and pagination
// in the options
//
// By default, progress is not included. Use `WithUserProgress()` on the options to include it
// By default, video metadata is not included. Use `WithAssetVideoMetadata()` on the options to include it
func (dao *DAO) ListAssets(ctx context.Context, dbOpts *Options) ([]*models.Asset, error) {
	builderOpts := newBuilderOptions(models.ASSET_TABLE).
		WithColumns(models.AssetColumns()...).
		SetDbOpts(dbOpts)

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress
	includeMetadata := dbOpts != nil && dbOpts.IncludeAssetMetadata
	includeCourse := dbOpts != nil && dbOpts.IncludeCourse
	includeLesson := dbOpts != nil && dbOpts.IncludeLesson

	// When no relations are included, use a simpler query
	if !includeProgress && !includeMetadata && !includeCourse && !includeLesson {
		return listGeneric[models.Asset](ctx, dao, *builderOpts)
	}

	// Progress join
	if includeProgress {
		principal, err := principalFromCtx(ctx)
		if err != nil {
			return nil, err
		}
		builderOpts = builderOpts.
			WithColumns(models.AssetProgressRowColumns()...).
			WithLeftJoin(
				models.ASSET_PROGRESS_TABLE,
				fmt.Sprintf(
					"%s = %s AND %s = '%s'",
					models.ASSET_PROGRESS_TABLE_ASSET_ID,
					models.ASSET_TABLE_ID,
					models.ASSET_PROGRESS_TABLE_USER_ID,
					principal.UserID,
				),
			)
	}

	// Metadata is fetched separately via GetAssetMetadataByAssetIDs when includeMetadata is true

	// Add lesson and course joins if enabled
	if dbOpts != nil {
		// If both course and lesson are requested, join lesson first, then course through lesson
		if dbOpts.IncludeCourse && dbOpts.IncludeLesson {
			builderOpts = builderOpts.
				WithJoin(models.LESSON_TABLE, models.ASSET_TABLE_LESSON_ID+" = "+models.LESSON_TABLE_ID).
				WithJoin(models.COURSE_TABLE, models.LESSON_TABLE_COURSE_ID+" = "+models.COURSE_TABLE_ID)
		} else if dbOpts.IncludeLesson {
			// Only lesson join
			builderOpts = builderOpts.WithJoin(models.LESSON_TABLE, models.ASSET_TABLE_LESSON_ID+" = "+models.LESSON_TABLE_ID)
		} else if dbOpts.IncludeCourse {
			// Only course join
			builderOpts = builderOpts.WithJoin(models.COURSE_TABLE, models.ASSET_TABLE_COURSE_ID+" = "+models.COURSE_TABLE_ID)
		}
	}

	rows, err := listGeneric[models.AssetRow](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	records := make([]*models.Asset, 0, len(rows))
	for i := range rows {
		records = append(records, rows[i].ToDomain(includeProgress, includeMetadata))
	}

	if includeMetadata && len(records) > 0 {
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
