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

var LogListApiAllowedFilters = []string{"level", "type", "component"}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateLog inserts a new log record
func (dao *DAO) CreateLog(ctx context.Context, log *models.Log) error {
	if log == nil {
		return utils.ErrNilPtr
	}

	if log.Message == "" {
		return utils.ErrLogMessage
	}

	if log.ID == "" {
		log.RefreshId()
	}

	log.RefreshCreatedAt()
	log.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.LOG_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:         log.ID,
				models.LOG_LEVEL:       log.Level,
				models.LOG_MESSAGE:     log.Message,
				models.LOG_DATA:        log.Data,
				models.BASE_CREATED_AT: log.CreatedAt,
				models.BASE_UPDATED_AT: log.UpdatedAt,
			},
		)

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetLog gets a record from the logs table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
func (dao *DAO) GetLog(ctx context.Context, dbOpts *Options) (*models.Log, error) {
	builderOpts := newBuilderOptions(models.LOG_TABLE).
		WithColumns(models.LogColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	return getGeneric[models.Log](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListLogs gets all records from the logs table based upon the where clause and pagination
// in the options
func (dao *DAO) ListLogs(ctx context.Context, dbOpts *Options) ([]*models.Log, error) {
	if err := parseLogApiQuery(dbOpts); err != nil {
		return nil, err
	}

	builderOpts := newBuilderOptions(models.LOG_TABLE).
		WithColumns(models.LogColumns()...).
		SetDbOpts(dbOpts)

	return listGeneric[models.Log](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateLogsBatch inserts multiple log records in a single query
func (dao *DAO) CreateLogsBatch(ctx context.Context, logs []*models.Log) error {
	if len(logs) == 0 {
		return nil
	}

	for _, log := range logs {
		if log == nil {
			return utils.ErrNilPtr
		}

		if log.Message == "" {
			return utils.ErrLogMessage
		}

		if log.ID == "" {
			log.RefreshId()
		}

		log.RefreshCreatedAt()
		log.RefreshUpdatedAt()
	}

	// Build batch insert query
	builder := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Question).
		Insert(models.LOG_TABLE).
		Columns(
			models.BASE_ID,
			models.LOG_LEVEL,
			models.LOG_MESSAGE,
			models.LOG_DATA,
			models.BASE_CREATED_AT,
			models.BASE_UPDATED_AT,
		)

	for _, log := range logs {
		builder = builder.Values(
			log.ID,
			log.Level,
			log.Message,
			log.Data,
			log.CreatedAt,
			log.UpdatedAt,
		)
	}

	sqlStr, args, err := builder.ToSql()
	if err != nil {
		return err
	}

	_, err = dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var defaultLogsListOrderBy = []string{models.LOG_TABLE_CREATED_AT + " desc", "rowid desc"}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseLogApiQuery parses dbOpts.ApiQuery and sets Where / OrderBy for log lists
func parseLogApiQuery(dbOpts *Options) error {
	if dbOpts == nil {
		return nil
	}

	q := dbOpts.ApiQuery

	if q == "" {
		if len(dbOpts.OrderBy) == 0 && dbOpts.OrderByClause == nil {
			dbOpts.WithOrderBy(defaultLogsListOrderBy...)
		}
		return nil
	}

	parsed, err := queryparser.Parse(q, LogListApiAllowedFilters)
	if err != nil {
		return fmt.Errorf("%w: %w", utils.ErrApiQueryParse, err)
	}

	if parsed == nil {
		dbOpts.WithOrderBy(defaultLogsListOrderBy...)
		return nil
	}

	if len(parsed.Sort) > 0 {
		dbOpts.WithOrderBy(parsed.Sort...)
	} else {
		dbOpts.WithOrderBy(defaultLogsListOrderBy...)
	}

	dbOpts.WithWhere(logWhereBuilder(parsed.Expr))
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// logWhereBuilder builds a squirrel WHERE expression from a queryparser.QueryExpr
func logWhereBuilder(expr queryparser.QueryExpr) squirrel.Sqlizer {
	switch node := expr.(type) {
	case *queryparser.ValueExpr:
		return squirrel.Like{models.LOG_TABLE_MESSAGE: "%" + node.Value + "%"}
	case *queryparser.FilterExpr:
		switch node.Key {
		case "level":
			return squirrel.Eq{models.LOG_TABLE_LEVEL: node.Value}
		case "type":
			return squirrel.Eq{"JSON_EXTRACT(" + models.LOG_TABLE_DATA + ", '$.type')": node.Value}
		case "component":
			return squirrel.Eq{"JSON_EXTRACT(" + models.LOG_TABLE_DATA + ", '$.component')": node.Value}
		default:
			return nil
		}
	case *queryparser.AndExpr:
		var andSlice []squirrel.Sqlizer
		for _, child := range node.Children {
			andSlice = append(andSlice, logWhereBuilder(child))
		}

		return squirrel.And(andSlice)
	case *queryparser.OrExpr:
		var orSlice []squirrel.Sqlizer
		for _, child := range node.Children {
			orSlice = append(orSlice, logWhereBuilder(child))
		}

		return squirrel.Or(orSlice)
	default:
		return nil
	}
}
