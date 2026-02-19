package dao

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateUser inserts a new user record
func (dao *DAO) CreateUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return utils.ErrNilPtr
	}

	if user.Username == "" {
		return utils.ErrUsername
	}

	if user.PasswordHash == "" {
		return utils.ErrUserPassword
	}

	if user.ID == "" {
		user.RefreshId()
	}

	user.RefreshCreatedAt()
	user.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.USER_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:            user.ID,
				models.USER_USERNAME:      user.Username,
				models.USER_DISPLAY_NAME:  user.DisplayName,
				models.USER_PASSWORD_HASH: user.PasswordHash,
				models.USER_ROLE:          user.Role,
				models.BASE_CREATED_AT:    user.CreatedAt,
				models.BASE_UPDATED_AT:    user.UpdatedAt,
			},
		)

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CountUsers counts the number of user records
func (dao *DAO) CountUsers(ctx context.Context, dbOpts *Options) (int, error) {
	builderOpts := newBuilderOptions(models.USER_TABLE).SetDbOpts(dbOpts)
	return countGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetUser gets a record from the user table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
func (dao *DAO) GetUser(ctx context.Context, dbOpts *Options) (*models.User, error) {
	builderOpts := newBuilderOptions(models.USER_TABLE).
		WithColumns(models.UserColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	return getGeneric[models.User](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListUsers gets all records from the user table based upon the where clause and pagination
// in the options
func (dao *DAO) ListUsers(ctx context.Context, dbOpts *Options) ([]*models.User, error) {
	builderOpts := newBuilderOptions(models.USER_TABLE).
		WithColumns(models.UserColumns()...).
		SetDbOpts(dbOpts)

	return listGeneric[models.User](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UpdateUser updates a user record
func (dao *DAO) UpdateUser(ctx context.Context, user *models.User) error {
	if user == nil {
		return utils.ErrNilPtr
	}

	if user.ID == "" {
		return utils.ErrId
	}

	if user.PasswordHash == "" {
		return utils.ErrUserPassword
	}

	user.RefreshUpdatedAt()

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: user.ID})

	builderOpts := newBuilderOptions(models.USER_TABLE).
		WithData(
			map[string]interface{}{
				models.USER_DISPLAY_NAME:  user.DisplayName,
				models.USER_PASSWORD_HASH: user.PasswordHash,
				models.USER_ROLE:          user.Role,
				models.BASE_UPDATED_AT:    user.UpdatedAt,
			},
		).
		SetDbOpts(dbOpts)

	_, err := updateGeneric(ctx, dao, *builderOpts)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteUsers deletes records from the user table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteUsers(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.USER_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}
