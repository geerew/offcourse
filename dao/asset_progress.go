package dao

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"text/template"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// assetProgressSuffixSQL is the ON CONFLICT DO UPDATE clause for asset progress upsert
//
// Rendered once at runtime
var assetProgressSuffixSQL = func() string {
	tmpl := template.Must(template.New("assetProgressSuffix").Parse(`
ON CONFLICT({{.AssetID}}, {{.UserID}}) DO UPDATE SET
  {{.Position}} = EXCLUDED.{{.Position}},
  {{.Completed}} = EXCLUDED.{{.Completed}},
  {{.CompletedAt}} = CASE
    WHEN {{.TableCompleted}} = 0 AND EXCLUDED.{{.Completed}} = 1 THEN STRFTIME('%Y-%m-%d %H:%M:%f','NOW')
    WHEN {{.TableCompleted}} = 1 AND EXCLUDED.{{.Completed}} = 0 THEN NULL
    ELSE {{.TableCompletedAt}}
  END,
  {{.ProgressFrac}} = EXCLUDED.{{.ProgressFrac}},
  {{.UpdatedAt}} = STRFTIME('%Y-%m-%d %H:%M:%f','NOW')
`))

	data := map[string]string{
		"AssetID":          models.ASSET_PROGRESS_ASSET_ID,
		"UserID":           models.ASSET_PROGRESS_USER_ID,
		"Position":         models.ASSET_PROGRESS_POSITION,
		"Completed":        models.ASSET_PROGRESS_COMPLETED,
		"CompletedAt":      models.ASSET_PROGRESS_COMPLETED_AT,
		"TableCompleted":   models.ASSET_PROGRESS_TABLE_COMPLETED,
		"TableCompletedAt": models.ASSET_PROGRESS_TABLE_COMPLETED_AT,
		"ProgressFrac":     models.ASSET_PROGRESS_PROGRESS_FRAC,
		"UpdatedAt":        models.BASE_UPDATED_AT,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic("assetProgressSuffix template: " + err.Error())
	}

	return buf.String()
}()

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// assetProgressFracSQL is the CASE expression for progress_frac
//
// Call with `squirrel.Expr(assetProgressFracSQL, isCompleted, assetID, position, assetID)`
//
// Placeholders:
//   - isCompleted,
//   - assetID,
//   - position,
//   - assetID
//
// Rendered once at runtime
var assetProgressFracSQL = func() string {
	tmpl := template.Must(template.New("assetProgressFrac").Parse(`
CASE
  WHEN ? = 1 THEN 1.0
  WHEN EXISTS (
    SELECT 1
    FROM {{.VideoTable}} v
    WHERE v.{{.VideoAssetID}} = ?
  )
  THEN MIN(
    1.0,
    (1.0 * ?) / NULLIF((
      SELECT v2.{{.VideoDuration}}
      FROM {{.VideoTable}} v2
      WHERE v2.{{.VideoAssetID}} = ?
    ), 0)
  )
  ELSE 0.0
END`))

	data := map[string]string{
		"VideoTable":    models.ASSET_METADATA_VIDEO_TABLE,
		"VideoAssetID":  models.ASSET_METADATA_VIDEO_ASSET_ID,
		"VideoDuration": models.ASSET_METADATA_VIDEO_DURATION,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		panic("assetProgressFrac template: " + err.Error())
	}
	return buf.String()
}()

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

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

	now := types.NowDateTime()
	createdAt := now

	completedAt := types.DateTime{}
	if assetProgress.Completed {
		completedAt = now
	}

	isCompleted := 0
	if assetProgress.Completed {
		isCompleted = 1
	}

	builder := newBuilderOptions(models.ASSET_PROGRESS_TABLE).
		WithSuffix(assetProgressSuffixSQL).
		WithData(map[string]interface{}{
			models.BASE_ID:                      assetProgress.ID,
			models.ASSET_PROGRESS_ASSET_ID:      assetProgress.AssetID,
			models.ASSET_PROGRESS_USER_ID:       assetProgress.UserID,
			models.ASSET_PROGRESS_POSITION:      assetProgress.Position,
			models.ASSET_PROGRESS_COMPLETED:     assetProgress.Completed,
			models.ASSET_PROGRESS_COMPLETED_AT:  completedAt,
			models.ASSET_PROGRESS_PROGRESS_FRAC: squirrel.Expr(assetProgressFracSQL, isCompleted, assetProgress.AssetID, assetProgress.Position, assetProgress.AssetID),
			models.BASE_CREATED_AT:              createdAt,
			models.BASE_UPDATED_AT:              now,
		})

	return dao.RunInTransaction(ctx, func(txCtx context.Context) error {
		if err := createGeneric(txCtx, dao, *builder); err != nil {
			if strings.HasPrefix(err.Error(), "FOREIGN KEY constraint failed") {
				return sql.ErrNoRows
			}
			return err
		}
		return dao.SyncCourseProgress(txCtx, assetProgress.AssetID)
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
