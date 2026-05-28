package dao

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateParam inserts a new param record
func (dao *DAO) CreateParam(ctx context.Context, param *models.Param) error {
	if param == nil {
		return utils.ErrNilPtr
	}

	if param.Key == "" {
		return utils.ErrKey
	}

	if param.ID == "" {
		param.RefreshId()
	}

	param.RefreshCreatedAt()
	param.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.PARAM_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:         param.ID,
				models.PARAM_KEY:       param.Key,
				models.PARAM_VALUE:     param.Value,
				models.BASE_CREATED_AT: param.CreatedAt,
				models.BASE_UPDATED_AT: param.UpdatedAt,
			},
		)

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetParam gets a record from the params table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
func (dao *DAO) GetParam(ctx context.Context, dbOpts *Options) (*models.Param, error) {
	builderOpts := newBuilderOptions(models.PARAM_TABLE).
		WithColumns(models.ParamColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	return getGeneric[models.Param](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListParams gets all records from the params table based upon the where clause and pagination
// in the options
func (dao *DAO) ListParams(ctx context.Context, dbOpts *Options) ([]*models.Param, error) {
	builderOpts := newBuilderOptions(models.PARAM_TABLE).
		WithColumns(models.ParamColumns()...).
		SetDbOpts(dbOpts)

	return listGeneric[models.Param](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UpdateParam updates a param record
func (dao *DAO) UpdateParam(ctx context.Context, param *models.Param) error {
	if param == nil {
		return utils.ErrNilPtr
	}

	if param.ID == "" {
		return utils.ErrId
	}

	if param.Key == "" {
		return utils.ErrKey
	}

	param.RefreshUpdatedAt()

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: param.ID})

	builderOpts := newBuilderOptions(models.PARAM_TABLE).
		WithData(
			map[string]interface{}{
				models.PARAM_VALUE:     param.Value,
				models.BASE_UPDATED_AT: param.UpdatedAt,
			},
		).
		SetDbOpts(dbOpts)

	_, err := updateGeneric(ctx, dao, *builderOpts)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteParams deletes records from the params table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteParams(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.PARAM_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}
