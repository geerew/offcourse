package dao

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateTag inserts a new tag record
func (dao *DAO) CreateTag(ctx context.Context, tag *models.Tag) error {
	if tag == nil {
		return utils.ErrNilPtr
	}

	if tag.Tag == "" {
		return utils.ErrTag
	}

	if tag.ID == "" {
		tag.RefreshId()
	}

	tag.RefreshCreatedAt()
	tag.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.TAG_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:         tag.ID,
				models.TAG_TAG:         tag.Tag,
				models.BASE_CREATED_AT: tag.CreatedAt,
				models.BASE_UPDATED_AT: tag.UpdatedAt,
			},
		)

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CountTags counts the number of tag records
func (dao *DAO) CountTags(ctx context.Context, dbOpts *Options) (int, error) {
	builderOpts := newBuilderOptions(models.TAG_TABLE).SetDbOpts(dbOpts)
	return countGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetTag gets a record from the tags table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
func (dao *DAO) GetTag(ctx context.Context, dbOpts *Options) (*models.Tag, error) {
	builderOpts := newBuilderOptions(models.TAG_TABLE).
		WithColumns(models.TagColumns()...).
		WithLeftJoin(models.COURSE_TAG_TABLE, fmt.Sprintf("%s = %s", models.COURSE_TAG_TABLE_TAG_ID, models.TAG_TABLE_ID)).
		WithGroupBy(models.TAG_TABLE_ID).
		SetDbOpts(dbOpts).
		WithLimit(1)

	return getGeneric[models.Tag](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListTags gets all records from the tags table based upon the where clause and pagination
// in the options
func (dao *DAO) ListTags(ctx context.Context, dbOpts *Options) ([]*models.Tag, error) {
	builderOpts := newBuilderOptions(models.TAG_TABLE).
		WithColumns(models.TagColumns()...).
		WithLeftJoin(models.COURSE_TAG_TABLE, fmt.Sprintf("%s = %s", models.COURSE_TAG_TABLE_TAG_ID, models.TAG_TABLE_ID)).
		WithGroupBy(models.TAG_TABLE_ID).
		SetDbOpts(dbOpts)

	return listGeneric[models.Tag](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListTagNames returns just the tag names as a []string
//
// TODO add tests
func (dao *DAO) ListTagNames(ctx context.Context, dbOpts *Options) ([]string, error) {
	builderOpts := newBuilderOptions(models.TAG_TABLE).
		WithColumns(models.TAG_TABLE + "." + models.TAG_TAG).
		SetDbOpts(dbOpts)

	return pluck[string](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UpdateTag updates a tag record
func (dao *DAO) UpdateTag(ctx context.Context, tag *models.Tag) error {
	if tag == nil {
		return utils.ErrNilPtr
	}

	if tag.ID == "" {
		return utils.ErrId
	}

	if tag.Tag == "" {
		return utils.ErrTag
	}

	tag.RefreshUpdatedAt()

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: tag.ID})

	builderOpts := newBuilderOptions(models.TAG_TABLE).
		WithData(
			map[string]interface{}{
				models.TAG_TAG:         tag.Tag,
				models.BASE_UPDATED_AT: tag.UpdatedAt,
			},
		).
		SetDbOpts(dbOpts)

	_, err := updateGeneric(ctx, dao, *builderOpts)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteTags deletes records from the tags table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteTags(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.TAG_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}
