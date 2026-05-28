package dao

import (
	"bytes"
	"context"
	"text/template"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/security"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// syncCourseProgressSQL recomputes a user's course progress and upserts it into
// courses_progress. The CTEs resolve the course from the given asset, then aggregate
// weighted progress across all assets in that course (percent, started_at, completed_at)
//
// Rendered once at runtime
var syncCourseProgressSQL = func() string {
	tmpl := template.Must(template.New("syncCourseProgress").Parse(`
WITH vars AS (
  SELECT (
    SELECT {{.AssetCourseID}}
    FROM {{.AssetTable}}
    WHERE {{.AssetID}} = ?
  ) AS course_id
),
ap AS (
  SELECT
    {{.AssetProgressAssetID}} AS asset_id,
    {{.AssetProgressPosition}} AS position,
    {{.AssetProgressCompleted}} AS completed,
    COALESCE({{.AssetProgressProgressFrac}}, 0.0) AS progress_frac,
    {{.AssetProgressCreatedAt}} AS created_at
  FROM {{.AssetProgressTable}}
  JOIN {{.AssetTable}}
    ON {{.AssetID}} = {{.AssetProgressAssetID}}
  WHERE {{.AssetCourseID}} = (SELECT course_id FROM vars)
    AND {{.AssetProgressUserID}} = ?
),
w AS (
  SELECT
    {{.AssetID}} AS asset_id,
    CASE WHEN {{.AssetWeight}} > 0 THEN {{.AssetWeight}} ELSE 1 END AS weight
  FROM {{.AssetTable}}
  WHERE {{.AssetCourseID}} = (SELECT course_id FROM vars)
),
totals AS (
  SELECT
    COALESCE(SUM(w.weight), 0) AS total_weight,
    COALESCE(SUM(ap.progress_frac * w.weight), 0.0) AS progress_weighted,
    MIN(CASE WHEN (ap.position > 0 OR ap.completed = 1) THEN ap.created_at END) AS started_at
  FROM w
  LEFT JOIN ap ON ap.asset_id = w.asset_id
)
INSERT INTO {{.CourseProgressTable}} (
  {{.BaseID}},
  {{.CourseProgressCourseID}},
  {{.CourseProgressUserID}},
  {{.CourseProgressStarted}},
  {{.CourseProgressStartedAt}},
  {{.CourseProgressPercent}},
  {{.CourseProgressCompletedAt}},
  {{.BaseCreatedAt}},
  {{.BaseUpdatedAt}}
)
VALUES (
  ?,
  (SELECT course_id FROM vars),
  ?,
  (SELECT CASE WHEN progress_weighted > 0 THEN 1 ELSE 0 END FROM totals),
  (SELECT started_at FROM totals),
  (SELECT CASE
            WHEN total_weight = 0 THEN 0
            ELSE CAST(ROUND(100.0 * progress_weighted / total_weight) AS INT)
          END
   FROM totals),
  (SELECT CASE
            WHEN total_weight > 0 AND progress_weighted >= total_weight
              THEN STRFTIME('%Y-%m-%d %H:%M:%f','NOW')
            ELSE NULL
          END
   FROM totals),
  STRFTIME('%Y-%m-%d %H:%M:%f','NOW'),
  STRFTIME('%Y-%m-%d %H:%M:%f','NOW')
)
ON CONFLICT({{.CourseProgressCourseID}}, {{.CourseProgressUserID}}) DO UPDATE SET
  {{.CourseProgressStarted}} = (SELECT CASE WHEN progress_weighted > 0 THEN 1 ELSE 0 END FROM totals),

  {{.CourseProgressStartedAt}} = CASE
         WHEN {{.CourseProgressStarted}} = 0 AND (SELECT started_at FROM totals) IS NOT NULL
           THEN (SELECT started_at FROM totals)
         ELSE {{.CourseProgressStartedAt}}
       END,

  {{.CourseProgressPercent}} = (SELECT CASE
                 WHEN total_weight = 0 THEN 0
                 ELSE CAST(ROUND(100.0 * progress_weighted / total_weight) AS INT)
               END
        FROM totals),

  {{.CourseProgressCompletedAt}} = CASE
         WHEN (SELECT total_weight FROM totals) > 0
              AND (SELECT progress_weighted FROM totals) >= (SELECT total_weight FROM totals)
              AND {{.CourseProgressCompletedAt}} IS NULL
           THEN STRFTIME('%Y-%m-%d %H:%M:%f','NOW')
         WHEN NOT (
               (SELECT total_weight FROM totals) > 0
               AND (SELECT progress_weighted FROM totals) >= (SELECT total_weight FROM totals)
              )
           THEN NULL
         ELSE {{.CourseProgressCompletedAt}}
       END,

  {{.BaseUpdatedAt}} = STRFTIME('%Y-%m-%d %H:%M:%f','NOW');
`))

	data := map[string]string{
		"AssetCourseID":             models.ASSET_TABLE_COURSE_ID,
		"AssetTable":                models.ASSET_TABLE,
		"AssetID":                   models.ASSET_TABLE_ID,
		"AssetWeight":               models.ASSET_TABLE_WEIGHT,
		"AssetProgressTable":        models.ASSET_PROGRESS_TABLE,
		"AssetProgressAssetID":      models.ASSET_PROGRESS_TABLE_ASSET_ID,
		"AssetProgressPosition":     models.ASSET_PROGRESS_TABLE_POSITION,
		"AssetProgressCompleted":    models.ASSET_PROGRESS_TABLE_COMPLETED,
		"AssetProgressProgressFrac": models.ASSET_PROGRESS_TABLE_PROGRESS_FRAC,
		"AssetProgressCreatedAt":    models.ASSET_PROGRESS_TABLE_CREATED_AT,
		"AssetProgressUserID":       models.ASSET_PROGRESS_TABLE_USER_ID,
		"CourseProgressTable":       models.COURSE_PROGRESS_TABLE,
		"BaseID":                    models.BASE_ID,
		"CourseProgressCourseID":    models.COURSE_PROGRESS_COURSE_ID,
		"CourseProgressUserID":      models.COURSE_PROGRESS_USER_ID,
		"CourseProgressStarted":     models.COURSE_PROGRESS_STARTED,
		"CourseProgressStartedAt":   models.COURSE_PROGRESS_STARTED_AT,
		"CourseProgressPercent":     models.COURSE_PROGRESS_PERCENT,
		"CourseProgressCompletedAt": models.COURSE_PROGRESS_COMPLETED_AT,
		"BaseCreatedAt":             models.BASE_CREATED_AT,
		"BaseUpdatedAt":             models.BASE_UPDATED_AT,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic("syncCourseProgress template: " + err.Error())
	}

	return buf.String()
}()

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SyncCourseProgressByAsset recomputes a user's course progress for the course
// that contains the given assetID
//
// If the asset doesn't exist, this is a no-op
func (dao *DAO) SyncCourseProgress(ctx context.Context, assetId string) error {
	if assetId == "" {
		return utils.ErrId
	}

	principal, err := principalFromCtx(ctx)
	if err != nil {
		return err
	}
	userID := principal.UserID

	args := []any{
		assetId,
		userID,
		security.PseudorandomString(10),
		userID,
	}

	_, err = dao.db.ExecContext(ctx, syncCourseProgressSQL, args...)
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
