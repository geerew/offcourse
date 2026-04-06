package dao

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/queryparser"
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
	if err := applyUsersStringQuery(dbOpts); err != nil {
		return nil, err
	}

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

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// applyUsersStringQuery parses dbOpts.StringQuery when Query is non-empty and sets Where / OrderBy.
func applyUsersStringQuery(dbOpts *Options) error {
	if dbOpts == nil || dbOpts.StringQuery == nil || dbOpts.StringQuery.Query == "" {
		return nil
	}

	parsed, err := queryparser.Parse(dbOpts.StringQuery.Query, dbOpts.StringQuery.AllowedFilters)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStringQueryParse, err)
	}

	if parsed == nil {
		return nil
	}

	if len(parsed.Sort) > 0 {
		dbOpts.OverrideOrderBy(parsed.Sort...)
	}

	dbOpts.WithWhere(usersWhereBuilder(parsed.Expr))

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// usersWhereBuilder builds a squirrel WHERE expression from a queryparser.QueryExpr
func usersWhereBuilder(expr queryparser.QueryExpr) squirrel.Sqlizer {
	switch node := expr.(type) {
	case *queryparser.ValueExpr:
		return squirrel.Or{
			squirrel.Like{"LOWER(" + models.USER_TABLE_USERNAME + ")": "%" + node.Value + "%"},
			squirrel.Like{"LOWER(" + models.USER_TABLE_DISPLAY_NAME + ")": "%" + node.Value + "%"},
		}
	case *queryparser.FilterExpr:
		switch node.Key {
		case "role":
			return squirrel.Eq{models.USER_TABLE_ROLE: node.Value}

		default:
			return nil
		}
	case *queryparser.AndExpr:
		var andSlice []squirrel.Sqlizer
		for _, child := range node.Children {
			andSlice = append(andSlice, usersWhereBuilder(child))
		}

		return squirrel.And(andSlice)
	case *queryparser.OrExpr:
		var orSlice []squirrel.Sqlizer
		for _, child := range node.Children {
			orSlice = append(orSlice, usersWhereBuilder(child))
		}

		return squirrel.Or(orSlice)
	default:
		return nil
	}
}
