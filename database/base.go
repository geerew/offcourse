package database

import (
	"context"
	"database/sql"

	"github.com/geerew/off-course/utils/appfs"
	"github.com/jmoiron/sqlx"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Database represents the database interface
type Database interface {
	// Querier methods
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error

	// Transaction methods
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error

	// DB methods
	DB() *sqlx.DB
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// txKey is the context key used to carry an active transaction
type txKey struct{}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// txFromContext returns the *sqlx.Tx stored in ctx, or nil
func txFromContext(ctx context.Context) *sqlx.Tx {
	if tx, ok := ctx.Value(txKey{}).(*sqlx.Tx); ok {
		return tx
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// withTx returns a context that carries for a transaction
func withTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DatabaseManagerConfig represents the settings needed to create a DatabaseManager
type DatabaseManagerConfig struct {
	// Where to write data.db & logs.db
	DataDir string

	// The application file system
	AppFs *appfs.AppFs

	// Whether to use an in-memory database
	Testing bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DatabaseManager manages different databases
type DatabaseManager struct {
	DataDb Database
	LogsDb Database
}
