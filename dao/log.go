package dao

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

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

	// Validate and prepare logs
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
