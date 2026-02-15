package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/geerew/off-course/migrations"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/security"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	migrateDirData = "data"
	migrateDirLogs = "logs"
	modeReadWrite  = "rwc"
	modeReadOnly   = "ro"
	dsnData        = "data.db"
	dsnLogs        = "logs.db"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var (
	gooseOnce sync.Once
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// sqliteConfig defines the configuration for a sqlite database
type sqliteConfig struct {
	// The directory where the database files are stored
	DataDir string

	// The name of the database file (ie data.db or logs.db)
	DSN string

	// The directory where the migration files are stored
	MigrateDir string

	// The application file system
	AppFs *appfs.AppFs

	// The database mode (ie read-only or read-write)
	Mode string

	// Whether to use an in-memory database (this is only used for testing)
	Testing bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewSQLiteManager returns a DatabaseManager
func NewSQLiteManager(config *DatabaseManagerConfig) (*DatabaseManager, error) {
	manager := &DatabaseManager{}

	dsnName := getDSNName(dsnData, config.Testing)

	writeCfg := &sqliteConfig{
		DataDir:    config.DataDir,
		DSN:        dsnName,
		MigrateDir: migrateDirData,
		AppFs:      config.AppFs,
		Testing:    config.Testing,
		Mode:       modeReadWrite,
	}

	writeDb, err := newSqliteDb(writeCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create write database: %w", err)
	}

	configureConnectionPool(writeDb, 1, 1)

	readCfg := &sqliteConfig{
		DataDir:    config.DataDir,
		DSN:        dsnName,
		MigrateDir: "",
		AppFs:      config.AppFs,
		Testing:    config.Testing,
		Mode:       modeReadOnly,
	}

	readDb, err := newSqliteDb(readCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create read database: %w", err)
	}

	configureConnectionPool(readDb, 10, 5)

	manager.DataDb = &sqliteCompositeDb{
		read:  readDb,
		write: writeDb,
	}

	dsnName = getDSNName(dsnLogs, config.Testing)

	logsCfg := &sqliteConfig{
		DataDir:    config.DataDir,
		DSN:        dsnName,
		MigrateDir: migrateDirLogs,
		AppFs:      config.AppFs,
		Testing:    config.Testing,
		Mode:       modeReadWrite,
	}

	logsDb, err := newSqliteDb(logsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs database: %w", err)
	}

	configureConnectionPool(logsDb, 1, 1)

	manager.LogsDb = logsDb

	return manager, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// sqliteCompositeDb - Read/write pools
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// sqliteCompositeDb represents a reader and writer db
type sqliteCompositeDb struct {
	read  *sqliteDb
	write *sqliteDb
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryContext executes a query that returns sql.Rows
//
// It implements the `Database` interface
func (c *sqliteCompositeDb) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return c.read.QueryContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryRowContext executes a query that returns a sql.Row
//
// It implements the `Database` interface
func (c *sqliteCompositeDb) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return c.read.QueryRowContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ExecContext executes a non-query SQL statement against the write pool, with automatic retry logic
// to handle SQLite lock contention. If the operation returns a "database is locked" or "
// table is locked" error, it will wait for an exponentially increasing backoff interval (up to
// defaultMaxLockRetries times) before retrying. Non-lock errors are returned immediately. If all
// retries fail, the final error is returned wrapped with the retry count. The retry logic respects
// context cancellation.
//
// It implements the Database interface
func (c *sqliteCompositeDb) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	var (
		res sql.Result
		err error
	)

	for attempt := 0; attempt <= defaultMaxLockRetries; attempt++ {
		// Check if context is cancelled before attempting
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}

		res, err = c.write.ExecContext(ctx, query, args...)
		if err == nil {
			return res, nil
		}

		// Bail on a non-lock error
		if !isLockError(err) {
			return res, err
		}

		delay := getRetryInterval(attempt)

		// On the last attempt, stop retrying
		if attempt == defaultMaxLockRetries {
			break
		}

		// Use context-aware sleep
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		case <-time.After(delay):
		}
	}

	return res, fmt.Errorf("%w after %d retries", err, defaultMaxLockRetries)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetContext retrieves a single row and scans it into dest (read pool)
//
// It implements the Querier interface
func (c *sqliteCompositeDb) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	return c.read.GetContext(ctx, dest, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SelectContext retrieves multiple rows and scans them into dest (read pool)
//
// It implements the Querier interface
func (c *sqliteCompositeDb) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return c.read.SelectContext(ctx, dest, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RunInTransaction runs a function in a transaction (write pool)
//
// It implements the Database interface
func (c *sqliteCompositeDb) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return c.write.RunInTransaction(ctx, fn)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DB returns the underlying sql.DB for the write pool
//
// It implements the Database interface
func (c *sqliteCompositeDb) DB() *sqlx.DB {
	return c.write.DB()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// sqliteTx - Transaction wrapper
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// sqliteTx is a sqlite-specific transaction wrapper
type sqliteTx struct {
	tx *sqlx.Tx
	db *sqliteDb
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ExecContext executes a query within a transaction without returning any rows
//
// It implements the Querier interface
func (tx *sqliteTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.tx.ExecContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryContext executes a query within a transaction that returns rows, typically a SELECT statement
//
// It implements the Querier interface
func (tx *sqliteTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.tx.QueryContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryRowContext executes a query within a transaction that is expected to return at most one row
//
// It implements the Querier interface
func (tx *sqliteTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.tx.QueryRowContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetContext retrieves a single row and scans it into dest
//
// It implements the Querier interface
func (tx *sqliteTx) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	return tx.tx.GetContext(ctx, dest, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SelectContext retrieves multiple rows and scans them into dest
//
// It implements the Querier interface
func (tx *sqliteTx) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return tx.tx.SelectContext(ctx, dest, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// sqliteDb - Single sqlite database
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// sqliteDb defines a sqlite database
type sqliteDb struct {
	sqlx   *sqlx.DB
	config *sqliteConfig
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// newSqliteDb creates a new sqliteDb
func newSqliteDb(config *sqliteConfig) (*sqliteDb, error) {
	db := &sqliteDb{
		config: config,
	}

	if err := db.bootstrap(); err != nil {
		return nil, err
	}

	if err := db.migrate(); err != nil {
		return nil, err
	}

	return db, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DB returns the underlying sql.DB
//
// It implements the Database interface
func (db *sqliteDb) DB() *sqlx.DB {
	return db.sqlx
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryContext executes a query that returns rows, typically a SELECT statement
//
// It implements the Database interface
func (db *sqliteDb) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.sqlx.QueryContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryRowContext executes a query that is expected to return at most one row
//
// It implements the Database interface
func (db *sqliteDb) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.sqlx.QueryRowContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ExecContext executes a query without returning any rows
//
// It implements the Database interface
func (db *sqliteDb) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.sqlx.ExecContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetContext retrieves a single row and scans it into dest
//
// It implements the Querier interface
func (db *sqliteDb) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	return db.sqlx.GetContext(ctx, dest, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SelectContext retrieves multiple rows and scans them into dest
//
// It implements the Querier interface
func (db *sqliteDb) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	return db.sqlx.SelectContext(ctx, dest, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RunInTransaction runs a function in a transaction with automatic retry logic
// for SQLite lock errors. If BeginTxx or operations inside the transaction return
// a "database is locked" or "table is locked" error, it will wait for an exponentially
// increasing backoff interval (up to defaultMaxLockRetries times) before retrying.
// Non-lock errors are returned immediately. If all retries fail, the final error is returned.
// The retry logic respects context cancellation.
//
// It implements the Database interface
func (db *sqliteDb) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	// Check if there's an existing transaction in the context
	if tx, ok := QuerierFromContext(ctx, nil).(*sqliteTx); ok && tx != nil {
		return fn(ctx)
	}

	var lastErr error

	for attempt := 0; attempt <= defaultMaxLockRetries; attempt++ {
		// Check if context is cancelled
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Begin transaction
		sqlxTx, err := db.sqlx.BeginTxx(ctx, nil)
		if err != nil {
			if !isLockError(err) {
				return err
			}
			lastErr = err
			if attempt < defaultMaxLockRetries {
				if err := sleepWithContext(ctx, getRetryInterval(attempt)); err != nil {
					return err
				}
				continue
			}
			break
		}

		// Execute function within transaction
		wrapped := &sqliteTx{tx: sqlxTx, db: db}
		txCtx := WithQuerier(ctx, wrapped)
		err = fn(txCtx)

		// Handle function result
		if err == nil {
			// Function succeeded - commit
			if commitErr := sqlxTx.Commit(); commitErr != nil {
				sqlxTx.Rollback()
				if isLockError(commitErr) && attempt < defaultMaxLockRetries {
					lastErr = commitErr
					if err := sleepWithContext(ctx, getRetryInterval(attempt)); err != nil {
						return err
					}
					continue
				}
				return commitErr
			}
			return nil
		}

		// Function failed - rollback
		sqlxTx.Rollback()
		lastErr = err

		// Retry if lock error, otherwise return immediately
		if !isLockError(err) || attempt >= defaultMaxLockRetries {
			return err
		}

		if err := sleepWithContext(ctx, getRetryInterval(attempt)); err != nil {
			return err
		}
	}

	return fmt.Errorf("%w after %d retries", lastErr, defaultMaxLockRetries)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// bootstrap initializes the sqlite database connection
func (db *sqliteDb) bootstrap() error {
	if err := db.config.AppFs.Fs.MkdirAll(db.config.DataDir, os.ModePerm); err != nil {
		return err
	}

	pragmaParts := []string{
		"cache=shared",
		"_busy_timeout=10000",
		"_journal_mode=WAL",
		"_journal_size_limit=200000000",
		"_synchronous=NORMAL",
		"_foreign_keys=1",
		"_cache_size=-16000",
	}

	if db.config.Mode != "" {
		pragmaParts = append([]string{fmt.Sprintf("mode=%s", db.config.Mode)}, pragmaParts...)
	}

	pragma := strings.Join(pragmaParts, "&")

	dsn := fmt.Sprintf("file:%s?%s", filepath.Join(db.config.DataDir, db.config.DSN), pragma)
	if db.config.Testing {
		dsn += "&mode=memory"
	}

	conn, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return err
	}

	conn.SetMaxIdleConns(1)
	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	db.sqlx = conn

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// migrate runs the goose migrations
func (db *sqliteDb) migrate() error {
	if db.config.MigrateDir == "" {
		return nil
	}

	gooseOnce.Do(func() {
		goose.SetLogger(goose.NopLogger())
		goose.SetBaseFS(migrations.EmbedMigrations)
		if err := goose.SetDialect("sqlite3"); err != nil {
			panic(fmt.Errorf("failed to set goose dialect: %w", err))
		}
	})

	if err := goose.Up(db.sqlx.DB, db.config.MigrateDir); err != nil {
		return fmt.Errorf("failed to run migrations in %s: %w", db.config.MigrateDir, err)
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// retry intervals for SQLite lock errors
var defaultRetryIntervals = []time.Duration{
	50 * time.Millisecond,
	100 * time.Millisecond,
	150 * time.Millisecond,
	200 * time.Millisecond,
	300 * time.Millisecond,
	400 * time.Millisecond,
	500 * time.Millisecond,
	700 * time.Millisecond,
	1000 * time.Millisecond,
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// how many times we’ll retry before giving up
const defaultMaxLockRetries = 9

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isLockError returns true for any SQLite “locked” error.
func isLockError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "database is locked") ||
		strings.Contains(s, "table is locked")
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// sleepWithContext sleeps for the given duration, respecting context cancellation
func sleepWithContext(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getRetryInterval picks a delay for the Nth retry
func getRetryInterval(attempt int) time.Duration {
	if attempt < 0 || attempt >= len(defaultRetryIntervals) {
		return defaultRetryIntervals[len(defaultRetryIntervals)-1]
	}
	return defaultRetryIntervals[attempt]
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getDSNName returns the name of the database file
//
// When testing is true, the database file name will be suffixed with a random string
// to avoid conflicts with other test cases
func getDSNName(baseName string, testing bool) string {
	if testing {
		return fmt.Sprintf("%s_memdb_%s", strings.TrimSuffix(baseName, ".db"), security.PseudorandomString(8))
	}
	return baseName
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// configureConnectionPool configures the connection pool for the sqlite database
func configureConnectionPool(db *sqliteDb, maxOpen, maxIdle int) {
	db.DB().SetMaxOpenConns(maxOpen)
	db.DB().SetMaxIdleConns(maxIdle)
	db.DB().SetConnMaxLifetime(time.Hour)
	db.DB().SetConnMaxIdleTime(10 * time.Minute)
}
