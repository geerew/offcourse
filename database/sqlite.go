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
	migrateDirData        = "data"
	migrateDirLogs        = "logs"
	modeReadWrite         = "rwc"
	modeReadOnly          = "ro"
	dsnData               = "data.db"
	dsnLogs               = "logs.db"
	defaultMaxLockRetries = 5
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var (
	gooseOnce             sync.Once
	defaultRetryIntervals = []time.Duration{
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

	writeDb, err := newSqliteConn(writeCfg)
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

	readDb, err := newSqliteConn(readCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create read database: %w", err)
	}

	configureConnectionPool(readDb, 10, 5)

	manager.DataDb = &SqliteDB{
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

	logsDb, err := newSqliteConn(logsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create logs database: %w", err)
	}

	configureConnectionPool(logsDb, 1, 1)

	manager.LogsDb = &SqliteDB{
		read:  logsDb,
		write: logsDb,
	}

	return manager, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SqliteDB represents a SQLite database connection with separate read and
// write pools
type SqliteDB struct {
	read  *sqlx.DB
	write *sqlx.DB
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ExecContext executes a non-query SQL statement
//
// It uses the write connection, supports transactions and retry logic for SQLite lock contention
func (db *SqliteDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.ExecContext(ctx, query, args...)
	}

	var (
		res sql.Result
		err error
	)

	for attempt := 0; attempt <= defaultMaxLockRetries; attempt++ {
		select {
		case <-ctx.Done():
			return res, ctx.Err()
		default:
		}

		res, err = db.write.ExecContext(ctx, query, args...)
		if err == nil {
			return res, nil
		}

		if !isLockError(err) {
			return res, err
		}

		if attempt == defaultMaxLockRetries {
			break
		}

		select {
		case <-ctx.Done():
			return res, ctx.Err()
		case <-time.After(getRetryInterval(attempt)):
		}
	}

	return res, fmt.Errorf("%w after %d retries", err, defaultMaxLockRetries)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryContext executes a query that returns sql.Rows
//
// It uses the read connection and supports transactions
func (db *SqliteDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryContext(ctx, query, args...)
	}

	return db.read.QueryContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryRowContext executes a query that returns a single sql.Row
//
// It uses the read connection and supports transactions
func (db *SqliteDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if tx := txFromContext(ctx); tx != nil {
		return tx.QueryRowContext(ctx, query, args...)
	}

	return db.read.QueryRowContext(ctx, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetContext retrieves a single row and automatically scans it into 'dest'
//
// It uses the read connection and supports transactions
func (db *SqliteDB) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	if tx := txFromContext(ctx); tx != nil {
		return tx.GetContext(ctx, dest, query, args...)
	}

	return db.read.GetContext(ctx, dest, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SelectContext retrieves multiple rows and automatically scans them into 'dest'
//
// It uses the read connection and supports transactions
func (db *SqliteDB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	if tx := txFromContext(ctx); tx != nil {
		return tx.SelectContext(ctx, dest, query, args...)
	}

	return db.read.SelectContext(ctx, dest, query, args...)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RunInTransaction runs the given function inside a transaction with automatic retry
// logic for SQLite lock errors
//
// Note: If the context already carries a transaction the function is executed directly
func (db *SqliteDB) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	// Just run 'fn' when already in a transaction
	if txFromContext(ctx) != nil {
		return fn(ctx)
	}

	var lastErr error

	for attempt := 0; attempt <= defaultMaxLockRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		sqlxTx, err := db.write.BeginTxx(ctx, nil)
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

		txCtx := withTx(ctx, sqlxTx)
		err = fn(txCtx)

		if err == nil {
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

		sqlxTx.Rollback()
		lastErr = err

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

// DB returns the underlying *sqlx.DB for the write pool.
func (db *SqliteDB) DB() *sqlx.DB {
	return db.write
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~=

// newSqliteConn bootstraps a single SQLite connection
func newSqliteConn(config *sqliteConfig) (*sqlx.DB, error) {
	if err := config.AppFs.Fs.MkdirAll(config.DataDir, os.ModePerm); err != nil {
		return nil, err
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

	if config.Mode != "" {
		pragmaParts = append([]string{fmt.Sprintf("mode=%s", config.Mode)}, pragmaParts...)
	}

	pragma := strings.Join(pragmaParts, "&")

	dsn := fmt.Sprintf("file:%s?%s", filepath.Join(config.DataDir, config.DSN), pragma)
	if config.Testing {
		dsn += "&mode=memory"
	}

	conn, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	conn.SetMaxIdleConns(1)
	conn.SetMaxOpenConns(1)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations if configured
	if config.MigrateDir != "" {
		if err := migrate(conn, config.MigrateDir); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// migrate runs the migrations for the given db, via goose
func migrate(db *sqlx.DB, migrateDir string) error {
	gooseOnce.Do(func() {
		goose.SetLogger(goose.NopLogger())
		goose.SetBaseFS(migrations.EmbedMigrations)
		if err := goose.SetDialect("sqlite3"); err != nil {
			panic(fmt.Errorf("failed to set goose dialect: %w", err))
		}
	})

	if err := goose.Up(db.DB, migrateDir); err != nil {
		return fmt.Errorf("failed to run migrations in %s: %w", migrateDir, err)
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isLockError returns true for any SQLite "locked" error
func isLockError(err error) bool {
	if err == nil {
		return false
	}

	s := err.Error()
	return strings.Contains(s, "database is locked") || strings.Contains(s, "table is locked")
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
func configureConnectionPool(db *sqlx.DB, maxOpen, maxIdle int) {
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)
}
