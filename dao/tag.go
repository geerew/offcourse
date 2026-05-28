package dao

import (
	"context"
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/queryparser"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var (
	TagListApiAllowedFilters = []string{"tag"}

	defaultTagsOrderBy = []string{models.TAG_TABLE_TAG + " asc"}
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
	if err := parseTagApiQuery(dbOpts); err != nil {
		return nil, err
	}

	applyDefaultOrderBy(dbOpts, defaultTagsOrderBy)

	builderOpts := newBuilderOptions(models.TAG_TABLE).
		WithColumns(models.TagColumns()...).
		WithLeftJoin(models.COURSE_TAG_TABLE, fmt.Sprintf("%s = %s", models.COURSE_TAG_TABLE_TAG_ID, models.TAG_TABLE_ID)).
		WithGroupBy(models.TAG_TABLE_ID).
		SetDbOpts(dbOpts)

	return listGeneric[models.Tag](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListTagNames returns the tag names as a string slice
//
// TODO add tests
func (dao *DAO) ListTagNames(ctx context.Context, dbOpts *Options) ([]string, error) {
	if err := parseTagApiQuery(dbOpts); err != nil {
		return nil, err
	}

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

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// tagOrderByIsSpecial checks if the order by clause is special
func tagOrderByIsSpecial(dbOpts *Options) bool {
	if dbOpts == nil || len(dbOpts.OrderBy) != 1 {
		return false
	}

	return strings.EqualFold(strings.TrimSpace(dbOpts.OrderBy[0]), "special")
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseTagApiQuery parses dbOpts.ApiQuery and sets a WHERE clause
func parseTagApiQuery(dbOpts *Options) (err error) {
	if dbOpts == nil {
		return nil
	}

	if dbOpts.ApiQuery == "" {
		if tagOrderByIsSpecial(dbOpts) {
			return fmt.Errorf("%w: orderBy special requires a search: filter in q", utils.ErrApiQueryParse)
		}

		return nil
	}

	parsed, parseErr := queryparser.Parse(dbOpts.ApiQuery, TagListApiAllowedFilters)
	if parseErr != nil {
		return fmt.Errorf("%w: %w", utils.ErrApiQueryParse, parseErr)
	}

	if parsed == nil || parsed.Expr == nil {
		if tagOrderByIsSpecial(dbOpts) {
			return fmt.Errorf("%w: orderBy special requires a search: filter in q", utils.ErrApiQueryParse)
		}

		return nil
	}

	if tagOrderByIsSpecial(dbOpts) {
		searchLower := firstSearchFilterLower(parsed.Expr)
		if searchLower == "" {
			return fmt.Errorf("%w: orderBy special requires a search: filter in q", utils.ErrApiQueryParse)
		}

		dbOpts.WithWhere(tagWhereBuilder(parsed.Expr))

		caseExpr := squirrel.Case().
			When(squirrel.Eq{"LOWER(" + models.TAG_TABLE_TAG + ")": searchLower}, "0").
			When(squirrel.Like{"LOWER(" + models.TAG_TABLE_TAG + ")": searchLower + "%"}, "1").
			When(squirrel.Like{"LOWER(" + models.TAG_TABLE_TAG + ")": "%" + searchLower + "%"}, "2")

		sql, args, _ := caseExpr.ToSql()
		dbOpts.OrderByClause = squirrel.Expr(sql+", "+defaultTagsOrderBy[0], args...)
		dbOpts.OrderBy = []string{}
	} else {
		dbOpts.WithWhere(tagWhereBuilder(parsed.Expr))
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// firstSearchFilterLower returns the first tag in the expression
func firstSearchFilterLower(expr queryparser.QueryExpr) string {
	if expr == nil {
		return ""
	}

	switch n := expr.(type) {
	case *queryparser.FilterExpr:
		if n.Key == "tag" {
			return strings.ToLower(n.Value)
		}
	case *queryparser.AndExpr:
		for _, c := range n.Children {
			if s := firstSearchFilterLower(c); s != "" {
				return s
			}
		}
	case *queryparser.OrExpr:
		for _, c := range n.Children {
			if s := firstSearchFilterLower(c); s != "" {
				return s
			}
		}
	}
	return ""
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// tagWhereBuilder builds a squirrel WHERE expression from a queryparser.QueryExpr
func tagWhereBuilder(expr queryparser.QueryExpr) squirrel.Sqlizer {
	switch node := expr.(type) {
	case *queryparser.FilterExpr:
		if node.Key == "tag" {
			return squirrel.Like{"LOWER(" + models.TAG_TABLE_TAG + ")": "%" + strings.ToLower(node.Value) + "%"}
		}

		return nil
	case *queryparser.AndExpr:
		var andSlice []squirrel.Sqlizer
		for _, child := range node.Children {
			andSlice = append(andSlice, tagWhereBuilder(child))
		}

		return squirrel.And(andSlice)
	case *queryparser.OrExpr:
		var orSlice []squirrel.Sqlizer
		for _, child := range node.Children {
			orSlice = append(orSlice, tagWhereBuilder(child))
		}

		return squirrel.Or(orSlice)
	default:
		return nil
	}
}
