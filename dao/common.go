package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/queryparser"
	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DAO is a data access object
type DAO struct {
	db database.Database
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// New creates a new DAO
func New(db database.Database) *DAO {
	return &DAO{db: db}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RunInTransaction is a wrapper for database.RunInTransaction
func (dao *DAO) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return dao.db.RunInTransaction(ctx, fn)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Create is a generic function to create a record in the database
func createGeneric(ctx context.Context, dao *DAO, builderOpts builderOptions) error {
	sqlStr, args, err := insertBuilder(builderOpts)
	if err != nil {
		return err
	}

	_, err = dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Count is a generic function to count the number of rows in a table as determined by the model
func countGeneric(ctx context.Context, dao *DAO, builderOpts builderOptions) (int, error) {
	sqlStr, args, err := countBuilder(builderOpts)
	if err != nil {
		return -1, err
	}

	var count int
	err = dao.db.GetContext(ctx, &count, sqlStr, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			count = 0
		} else {
			return -1, err
		}
	}

	return count, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getGeneric is a generic function to get a record from the database
func getGeneric[T any](ctx context.Context, dao *DAO, builderOpts builderOptions) (*T, error) {
	sqlStr, args, err := selectBuilder(builderOpts)
	if err != nil {
		return nil, err
	}

	record := new(T)
	err = dao.db.GetContext(ctx, record, sqlStr, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return record, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// listGeneric is a generic function to get records from the database
func listGeneric[T any](ctx context.Context, dao *DAO, builderOpts builderOptions) ([]*T, error) {
	if builderOpts.DbOpts != nil && builderOpts.DbOpts.Pagination != nil {
		count, err := countGeneric(ctx, dao, builderOpts)
		if err != nil {
			return nil, err
		}

		builderOpts.DbOpts.Pagination.SetCount(count)
	}

	sqlStr, args, err := selectBuilder(builderOpts)
	if err != nil {
		return nil, err
	}

	records := new([]*T)
	err = dao.db.SelectContext(ctx, records, sqlStr, args...)
	if err != nil {
		return nil, err
	}

	return *records, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// updateGeneric is a generic function to update a record in the database
func updateGeneric(ctx context.Context, dao *DAO, builderOpts builderOptions) (bool, error) {
	sqlStr, args, err := updateBuilder(builderOpts)
	if err != nil {
		return false, err
	}

	res, err := dao.db.ExecContext(ctx, sqlStr, args...)
	if err != nil {
		return false, err
	}

	rowCount, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	return rowCount > 0, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// pluck returns a slice of a single column (e.g., []string, []int)
// from the given table using the same Options/joins/filters you already use.
// Pass column as a fully-qualified name (e.g., "tags.tag").
// If distinct is true, it will SELECT DISTINCT <column>.
func pluck[T any](ctx context.Context, dao *DAO, builderOpts builderOptions) ([]T, error) {
	sqlStr, args, err := selectBuilder(builderOpts)
	if err != nil {
		return nil, err
	}

	var out []T
	if err := dao.db.SelectContext(ctx, &out, sqlStr, args...); err != nil {
		return nil, err
	}

	return out, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// principalFromCtx is a helper method to get the principal from the context
func principalFromCtx(ctx context.Context) (types.Principal, error) {
	principal, ok := ctx.Value(types.PrincipalContextKey).(types.Principal)
	if !ok {
		return types.Principal{}, utils.ErrPrincipal
	}

	return principal, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func sanitizeIDs(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, id := range in {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ErrStringQueryParse is returned when the `q` parameter fails to parse
var ErrStringQueryParse = errors.New("list query parse error")

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// applyStringQuerySortOnly parses dbOpts.StringQuery when Query is non-empty and applies sort tokens only
// (no WHERE from the expression). Used for lesson and attachment lists that support `q` ordering.
func applyStringQuerySortOnly(dbOpts *Options) error {
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

	return nil
}
