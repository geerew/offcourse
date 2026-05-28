package dao

import (
	"context"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateCourseFavourite inserts a new course favourite record
func (dao *DAO) CreateCourseFavourite(ctx context.Context, courseFavourite *models.CourseFavourite) error {
	if courseFavourite == nil {
		return utils.ErrNilPtr
	}

	if courseFavourite.CourseID == "" {
		return utils.ErrCourseId
	}

	if courseFavourite.UserID == "" {
		return utils.ErrUserId
	}

	if courseFavourite.ID == "" {
		courseFavourite.RefreshId()
	}

	courseFavourite.RefreshCreatedAt()
	courseFavourite.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.COURSE_FAVOURITE_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:                    courseFavourite.ID,
				models.COURSE_FAVOURITE_COURSE_ID: courseFavourite.CourseID,
				models.COURSE_FAVOURITE_USER_ID:   courseFavourite.UserID,
				models.BASE_CREATED_AT:            courseFavourite.CreatedAt,
				models.BASE_UPDATED_AT:            courseFavourite.UpdatedAt,
			},
		)

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetCourseFavourite gets a record from the course favourites table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
func (dao *DAO) GetCourseFavourite(ctx context.Context, dbOpts *Options) (*models.CourseFavourite, error) {
	builderOpts := newBuilderOptions(models.COURSE_FAVOURITE_TABLE).
		WithColumns(models.CourseFavouriteColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	return getGeneric[models.CourseFavourite](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListCourseFavourites gets all records from the course favourites table based upon the where clause and pagination
// in the options
func (dao *DAO) ListCourseFavourites(ctx context.Context, dbOpts *Options) ([]*models.CourseFavourite, error) {
	builderOpts := newBuilderOptions(models.COURSE_FAVOURITE_TABLE).
		WithColumns(models.CourseFavouriteColumns()...).
		SetDbOpts(dbOpts)

	return listGeneric[models.CourseFavourite](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteCourseFavourites deletes records from the course favourites table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteCourseFavourites(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.COURSE_FAVOURITE_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}
