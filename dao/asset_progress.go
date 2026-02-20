package dao

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/types"
)

// UpsertAssetProgress upserts an asset progress record for a user
func (dao *DAO) UpsertAssetProgress(ctx context.Context, assetProgress *models.AssetProgress) error {
	if assetProgress == nil {
		return utils.ErrNilPtr
	}

	if assetProgress.AssetID == "" {
		return utils.ErrId
	}

	principal, err := principalFromCtx(ctx)
	if err != nil {
		return err
	}
	assetProgress.UserID = principal.UserID

	if assetProgress.ID == "" {
		assetProgress.RefreshId()
	}

	// Build the upsert query. This will insert a new record or update an existing one
	now := types.NowDateTime()
	createdAt := now

	completedAt := types.DateTime{}
	if assetProgress.Completed {
		completedAt = now
	}

	upsertBuilder := newBuilderOptions(models.ASSET_PROGRESS_TABLE).
		WithData(map[string]interface{}{
			models.BASE_ID:                     assetProgress.ID,
			models.ASSET_PROGRESS_ASSET_ID:     assetProgress.AssetID,
			models.ASSET_PROGRESS_USER_ID:      assetProgress.UserID,
			models.ASSET_PROGRESS_POSITION:     assetProgress.Position,
			models.ASSET_PROGRESS_COMPLETED:    assetProgress.Completed,
			models.ASSET_PROGRESS_COMPLETED_AT: completedAt,
			models.BASE_CREATED_AT:             createdAt,
			models.BASE_UPDATED_AT:             now,
		}).
		WithSuffix(upsertAssetProgressSuffix())

	// Build the progress fraction update query. This will always update the progress_frac column
	dbOpts := NewOptions().WithWhere(squirrel.And{
		squirrel.Eq{models.ASSET_PROGRESS_ASSET_ID: assetProgress.AssetID},
		squirrel.Eq{models.ASSET_PROGRESS_USER_ID: assetProgress.UserID},
	})

	progressFracBuilder := newBuilderOptions(models.ASSET_PROGRESS_TABLE).
		WithData(map[string]interface{}{
			models.ASSET_PROGRESS_PROGRESS_FRAC: progressFracCaseExpr(),
		}).
		SetDbOpts(dbOpts)

	return dao.db.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Upsert position, completed, completed_at, updated_at
		if err := createGeneric(txCtx, dao, *upsertBuilder); err != nil {
			if strings.HasPrefix(err.Error(), "FOREIGN KEY constraint failed") {
				return sql.ErrNoRows
			}

			return err
		}

		// Update progress_frac
		if _, err := updateGeneric(txCtx, dao, *progressFracBuilder); err != nil {
			return err
		}

		// Sync course progress
		err := dao.SyncCourseProgress(txCtx, assetProgress.AssetID)
		return err
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAssetProgress gets a record from the asset progress table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
func (dao *DAO) GetAssetProgress(ctx context.Context, dbOpts *Options) (*models.AssetProgress, error) {
	builderOpts := newBuilderOptions(models.ASSET_PROGRESS_TABLE).
		WithColumns(models.AssetProgressColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	return getGeneric[models.AssetProgress](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListAssetProgress gets all records from the asset progress table based upon the where clause and pagination
// in the options
func (dao *DAO) ListAssetProgress(ctx context.Context, dbOpts *Options) ([]*models.AssetProgress, error) {
	builderOpts := newBuilderOptions(models.ASSET_PROGRESS_TABLE).
		WithColumns(models.AssetProgressColumns()...).
		SetDbOpts(dbOpts)

	return listGeneric[models.AssetProgress](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// ListAssetProgressIDs returns just the asset progress IDs as a []string
//
// TODO add tests
func (dao *DAO) ListAssetProgressIDs(ctx context.Context, dbOpts *Options) ([]string, error) {
	builderOpts := newBuilderOptions(models.ASSET_PROGRESS_TABLE).
		WithColumns(models.ASSET_PROGRESS_TABLE_ID).
		SetDbOpts(dbOpts)

	return pluck[string](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteAssetProgress deletes records from the asset progress table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteAssetProgress(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.ASSET_PROGRESS_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteAssetProgressForCourse deletes all asset progress records for a given user that
// belong to a course
func (dao *DAO) DeleteAssetProgressForCourse(ctx context.Context, courseID, userID string) error {
	if courseID == "" {
		return utils.ErrCourseId
	}

	if userID == "" {
		return utils.ErrUserId
	}

	where := squirrel.And{
		squirrel.Eq{models.ASSET_PROGRESS_TABLE_USER_ID: userID},
		squirrel.Expr(
			"EXISTS (SELECT 1 FROM "+models.ASSET_TABLE+
				" WHERE "+models.ASSET_TABLE_ID+" = "+models.ASSET_PROGRESS_TABLE_ASSET_ID+
				" AND "+models.ASSET_TABLE_COURSE_ID+" = ?)",
			courseID,
		),
	}

	dbOpts := NewOptions().WithWhere(where)

	return dao.DeleteAssetProgress(ctx, dbOpts)
}

// ~~~ helpers ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Builds a query to upsert the asset progress, without updating progress_frac
func upsertAssetProgressSuffix() string {
	return fmt.Sprintf(`
ON CONFLICT(%s, %s) DO UPDATE SET
  -- Position
  %s = EXCLUDED.%s,

  -- Completed flag
  %s = EXCLUDED.%s,

  -- Completed timestamp (edge-aware)
  %s = CASE
          WHEN %s.%s = 0 AND EXCLUDED.%s = 1 THEN STRFTIME('%%Y-%%m-%%d %%H:%%M:%%f','NOW')
          WHEN %s.%s = 1 AND EXCLUDED.%s = 0 THEN NULL
          ELSE %s.%s
        END,

  -- Always bump updated_at
  %s = STRFTIME('%%Y-%%m-%%d %%H:%%M:%%f','NOW')
`,
		// conflict target
		models.ASSET_PROGRESS_ASSET_ID, models.ASSET_PROGRESS_USER_ID,

		// position
		models.ASSET_PROGRESS_POSITION, models.ASSET_PROGRESS_POSITION,

		// completed
		models.ASSET_PROGRESS_COMPLETED, models.ASSET_PROGRESS_COMPLETED,

		// completed_at CASE
		models.ASSET_PROGRESS_COMPLETED_AT,
		models.ASSET_PROGRESS_TABLE, models.ASSET_PROGRESS_COMPLETED, models.ASSET_PROGRESS_COMPLETED,
		models.ASSET_PROGRESS_TABLE, models.ASSET_PROGRESS_COMPLETED, models.ASSET_PROGRESS_COMPLETED,
		models.ASSET_PROGRESS_TABLE, models.ASSET_PROGRESS_COMPLETED_AT,

		// updated_at
		models.BASE_UPDATED_AT,
	)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Computes progress_frac purely from server-side data.
//   - completed = 1 => 1.0
//   - if there is video metadata => position/duration clamped to 1.0
//   - else (non-video or unknown duration) => 0.0
func progressFracCaseExpr() squirrel.Sqlizer {
	return squirrel.Expr(fmt.Sprintf(`
CASE
  WHEN %s = 1 THEN 1.0
  WHEN EXISTS (
    SELECT 1
    FROM %s v
    WHERE v.%s = %s.%s
  )
  THEN MIN(
    1.0,
    (1.0 * %s) / NULLIF((
      SELECT v2.%s
      FROM %s v2
      WHERE v2.%s = %s.%s
    ), 0)
  )
  ELSE 0.0
END`,
		// completed
		models.ASSET_PROGRESS_COMPLETED,

		// EXISTS asset_media_video
		models.ASSET_METADATA_VIDEO_TABLE,
		models.ASSET_METADATA_VIDEO_ASSET_ID, models.ASSET_PROGRESS_TABLE, models.ASSET_PROGRESS_ASSET_ID,

		// numerator: position
		models.ASSET_PROGRESS_POSITION,

		// denominator: duration_sec subselect
		models.ASSET_METADATA_VIDEO_DURATION,
		models.ASSET_METADATA_VIDEO_TABLE,
		models.ASSET_METADATA_VIDEO_ASSET_ID, models.ASSET_PROGRESS_TABLE, models.ASSET_PROGRESS_ASSET_ID,
	))
}
