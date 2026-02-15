package database

import (
	"context"
	"database/sql"

	"github.com/geerew/off-course/utils/appfs"
	"github.com/jmoiron/sqlx"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type contextKey string

const querierKey = contextKey("querier")

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithQuerier adds a querier to the context
func WithQuerier(ctx context.Context, querier Querier) context.Context {
	return context.WithValue(ctx, querierKey, querier)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QuerierFromContext returns the querier from the context, or defaultQuerier if not found
func QuerierFromContext(ctx context.Context, defaultQuerier Querier) Querier {
	if querier, ok := ctx.Value(querierKey).(Querier); ok && querier != nil {
		return querier
	}

	return defaultQuerier
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Database defines the interface for a database
type Database interface {
	Querier
	DB() *sqlx.DB
	RunInTransaction(context.Context, func(context.Context) error) error
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Querier defines the interface for a DB querier
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row

	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DatabaseManagerConfig holds only the settings needed to create a new DatabaseManager
type DatabaseManagerConfig struct {
	// Where to write data.db & logs.db
	DataDir string

	// The application file system
	AppFs *appfs.AppFs

	// Whether to use an in-memory database (this is only used for testing)
	Testing bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DatabaseManager manages the database connections
type DatabaseManager struct {
	DataDb Database
	LogsDb Database
}
