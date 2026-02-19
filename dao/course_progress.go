package dao

import (
	"context"
	"fmt"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/security"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SyncCourseProgressByAsset recomputes a user's course progress for the course
// that contains the given assetID. If the asset doesn't exist, this is a no-op.
func (dao *DAO) SyncCourseProgress(ctx context.Context, assetId string) error {
	if assetId == "" {
		return utils.ErrId
	}

	principal, err := principalFromCtx(ctx)
	if err != nil {
		return err
	}
	userID := principal.UserID

	sqlSync := buildSyncCourseProgressSQL()

	args := []any{
		assetId,
		userID,
		security.PseudorandomString(10),
		userID,
	}

	_, err = dao.db.ExecContext(ctx, sqlSync, args...)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetCourseProgress gets a record from the course progress table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
func (dao *DAO) GetCourseProgress(ctx context.Context, dbOpts *Options) (*models.CourseProgress, error) {
	builderOpts := newBuilderOptions(models.COURSE_PROGRESS_TABLE).
		WithColumns(models.CourseProgressColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	return getGeneric[models.CourseProgress](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListCourseProgress gets all records from the course progress table based upon the where clause and pagination
// in the options
func (dao *DAO) ListCourseProgress(ctx context.Context, dbOpts *Options) ([]*models.CourseProgress, error) {
	builderOpts := newBuilderOptions(models.COURSE_PROGRESS_TABLE).
		WithColumns(models.CourseProgressColumns()...).
		SetDbOpts(dbOpts)

	return listGeneric[models.CourseProgress](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteCourseProgress deletes records from the course progress table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteCourseProgress(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.COURSE_PROGRESS_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// ~~~ helpers ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// buildSyncCourseProgressSQL returns the CTE+UPSERT query for course progress
func buildSyncCourseProgressSQL() string {
	return fmt.Sprintf(`
WITH vars AS (
  SELECT (
    SELECT %s
    FROM %s
    WHERE %s = ?
  ) AS course_id
),
ap AS (
  SELECT
    %s AS asset_id,
    %s AS position,
    %s AS completed,
    COALESCE(%s, 0.0) AS progress_frac,
    %s AS created_at
  FROM %s
  JOIN %s
    ON %s = %s
  WHERE %s = (SELECT course_id FROM vars)
    AND %s = ?
),
w AS (
  SELECT
    %s AS asset_id,
    CASE WHEN %s > 0 THEN %s ELSE 1 END AS weight
  FROM %s
  WHERE %s = (SELECT course_id FROM vars)
),
totals AS (
  SELECT
    COALESCE(SUM(w.weight), 0) AS total_weight,
    COALESCE(SUM(ap.progress_frac * w.weight), 0.0) AS progress_weighted,
    MIN(CASE WHEN (ap.position > 0 OR ap.completed = 1) THEN ap.created_at END) AS started_at
  FROM w
  LEFT JOIN ap ON ap.asset_id = w.asset_id
)
INSERT INTO %s (
  %s,  -- id
  %s,  -- course_id
  %s,  -- user_id
  %s,  -- started
  %s,  -- started_at
  %s,  -- percent
  %s,  -- completed_at
  %s,  -- created_at
  %s   -- updated_at
)
VALUES (
  ?,  -- id
  (SELECT course_id FROM vars),  -- course_id
  ?,  -- user_id
  (SELECT CASE WHEN progress_weighted > 0 THEN 1 ELSE 0 END FROM totals),
  (SELECT started_at FROM totals),
  (SELECT CASE
            WHEN total_weight = 0 THEN 0
            ELSE CAST(ROUND(100.0 * progress_weighted / total_weight) AS INT)
          END
   FROM totals),
  -- set completed_at on insert if already 100
  (SELECT CASE
            WHEN total_weight > 0 AND progress_weighted >= total_weight
              THEN STRFTIME('%%Y-%%m-%%d %%H:%%M:%%f','NOW')
            ELSE NULL
          END
   FROM totals),
  STRFTIME('%%Y-%%m-%%d %%H:%%M:%%f','NOW'),
  STRFTIME('%%Y-%%m-%%d %%H:%%M:%%f','NOW')
)
ON CONFLICT(%s, %s) DO UPDATE SET
  %s = (SELECT CASE WHEN progress_weighted > 0 THEN 1 ELSE 0 END FROM totals),

  %s = CASE
         WHEN %s = 0 AND (SELECT started_at FROM totals) IS NOT NULL
           THEN (SELECT started_at FROM totals)
         ELSE %s
       END,

  %s = (SELECT CASE
                 WHEN total_weight = 0 THEN 0
                 ELSE CAST(ROUND(100.0 * progress_weighted / total_weight) AS INT)
               END
        FROM totals),

  %s = CASE
         WHEN (SELECT total_weight FROM totals) > 0
              AND (SELECT progress_weighted FROM totals) >= (SELECT total_weight FROM totals)
              AND %s IS NULL
           THEN STRFTIME('%%Y-%%m-%%d %%H:%%M:%%f','NOW')
         WHEN NOT (
               (SELECT total_weight FROM totals) > 0
               AND (SELECT progress_weighted FROM totals) >= (SELECT total_weight FROM totals)
              )
           THEN NULL
         ELSE %s
       END,

  %s = STRFTIME('%%Y-%%m-%%d %%H:%%M:%%f','NOW');
`,
		// vars: SELECT course_id FROM assets WHERE id = ?
		models.ASSET_TABLE_COURSE_ID, // assets.course_id
		models.ASSET_TABLE,           // assets
		models.ASSET_TABLE_ID,        // assets.id

		// ap: select list (from assets_progress joined to assets)
		models.ASSET_PROGRESS_TABLE_ASSET_ID,
		models.ASSET_PROGRESS_TABLE_POSITION,
		models.ASSET_PROGRESS_TABLE_COMPLETED,
		models.ASSET_PROGRESS_TABLE_PROGRESS_FRAC,
		models.ASSET_PROGRESS_TABLE_CREATED_AT,
		models.ASSET_PROGRESS_TABLE,          // FROM assets_progress
		models.ASSET_TABLE,                   // JOIN assets
		models.ASSET_TABLE_ID,                // assets.id
		models.ASSET_PROGRESS_TABLE_ASSET_ID, // assets_progress.asset_id
		models.ASSET_TABLE_COURSE_ID,         // WHERE assets.course_id = vars.course_id
		models.ASSET_PROGRESS_TABLE_USER_ID,  // AND assets_progress.user_id = ?

		// w: weights from assets
		models.ASSET_TABLE_ID,
		models.ASSET_TABLE_WEIGHT,
		models.ASSET_TABLE_WEIGHT,
		models.ASSET_TABLE,
		models.ASSET_TABLE_COURSE_ID,

		// INSERT INTO courses_progress (...)
		models.COURSE_PROGRESS_TABLE,
		models.BASE_ID,
		models.COURSE_PROGRESS_COURSE_ID,
		models.COURSE_PROGRESS_USER_ID,
		models.COURSE_PROGRESS_STARTED,
		models.COURSE_PROGRESS_STARTED_AT,
		models.COURSE_PROGRESS_PERCENT,
		models.COURSE_PROGRESS_COMPLETED_AT,
		models.BASE_CREATED_AT,
		models.BASE_UPDATED_AT,

		// ON CONFLICT keys
		models.COURSE_PROGRESS_COURSE_ID,
		models.COURSE_PROGRESS_USER_ID,

		// UPDATE SET
		models.COURSE_PROGRESS_STARTED,
		models.COURSE_PROGRESS_STARTED_AT,
		models.COURSE_PROGRESS_STARTED,
		models.COURSE_PROGRESS_STARTED_AT,
		models.COURSE_PROGRESS_PERCENT,
		models.COURSE_PROGRESS_COMPLETED_AT,
		models.COURSE_PROGRESS_COMPLETED_AT,
		models.COURSE_PROGRESS_COMPLETED_AT,
		models.BASE_UPDATED_AT,
	)
}
