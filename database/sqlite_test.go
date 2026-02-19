package database

import (
	"context"
	"testing"

	"github.com/geerew/off-course/utils/appfs"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func setupSqliteDB(t *testing.T) *DatabaseManager {
	t.Helper()

	appFs := appfs.New(afero.NewMemMapFs())

	dbManager, err := NewSQLiteManager(&DatabaseManagerConfig{
		DataDir: "./oc_data",
		AppFs:   appFs,
		Testing: true,
	})

	require.NoError(t, err)
	require.NotNil(t, dbManager)

	// Create a test table
	_, err = dbManager.DataDb.ExecContext(context.Background(), "CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)

	return dbManager
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqliteDb_Bootstrap(t *testing.T) {
	// Test successfully creating a sqlite connection
	t.Run("success", func(t *testing.T) {

		appFs := appfs.New(afero.NewMemMapFs())

		db, err := newSqliteConn(&sqliteConfig{
			DataDir:    "./oc_data",
			DSN:        "data.db",
			MigrateDir: "data",
			AppFs:      appFs,
			Testing:    true,
		})

		require.NoError(t, err)
		require.NotNil(t, db)

	})

	// Test error due to being unable to create data.db
	t.Run("error creating data.db", func(t *testing.T) {
		appFs := appfs.New(afero.NewReadOnlyFs(afero.NewMemMapFs()))

		db, err := newSqliteConn(&sqliteConfig{
			DataDir:    "./oc_data",
			DSN:        "data.db",
			MigrateDir: "data",
			AppFs:      appFs,
			Testing:    true,
		})

		require.NotNil(t, err)
		require.EqualError(t, err, "operation not permitted")
		require.Nil(t, db)
	})

	// Test error due to invalid migration directory
	t.Run("invalid migration", func(t *testing.T) {
		appFs := appfs.New(afero.NewMemMapFs())

		db, err := newSqliteConn(&sqliteConfig{
			DataDir:    "./oc_data",
			DSN:        "data.db",
			MigrateDir: "test",
			AppFs:      appFs,
			Testing:    true,
		})

		require.NotNil(t, err)
		require.Contains(t, err.Error(), "failed to run migrations in test")
		require.Contains(t, err.Error(), "test directory does not exist")
		require.Nil(t, db)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqliteDb_QueryContext(t *testing.T) {
	// Test successfully querying multiple rows
	t.Run("simple", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		_, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test')")
		require.NoError(t, err)

		rows, err := dbManager.DataDb.QueryContext(ctx, "SELECT * FROM test")
		require.NoError(t, err)
		defer rows.Close()

		for rows.Next() {
			var id int
			var name string
			err = rows.Scan(&id, &name)
			require.NoError(t, err)
			require.Equal(t, 1, id)
			require.Equal(t, "test", name)
		}

		require.Nil(t, rows.Err())
	})

	//

	// Test successfully querying multiple rows in a transaction
	t.Run("transaction", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		_, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test')")
		require.NoError(t, err)

		var id int
		var name string

		err = dbManager.DataDb.RunInTransaction(ctx, func(txCtx context.Context) error {
			rows, err := dbManager.DataDb.QueryContext(txCtx, "SELECT * FROM test")
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				err = rows.Scan(&id, &name)
				require.NoError(t, err)
			}

			return nil
		})

		require.NoError(t, err)
		require.Equal(t, "test", name)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqliteDb_QueryRowContext(t *testing.T) {
	// Test successfully querying a single row
	t.Run("simple", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		_, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test')")
		require.NoError(t, err)

		var id int
		var name string
		err = dbManager.DataDb.QueryRowContext(ctx, "SELECT * FROM test").Scan(&id, &name)

		require.NoError(t, err)
		require.Equal(t, "test", name)
	})

	// Test successfully querying a single row in a transaction
	t.Run("transaction", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		_, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test')")
		require.NoError(t, err)

		var id int
		var name string

		err = dbManager.DataDb.RunInTransaction(ctx, func(txCtx context.Context) error {
			return dbManager.DataDb.QueryRowContext(txCtx, "SELECT * FROM test").Scan(&id, &name)
		})

		require.NoError(t, err)
		require.Equal(t, "test", name)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqliteDb_ExecContext(t *testing.T) {
	// Test successfully executing a non-query SQL statement
	t.Run("simple", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		result, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test')")
		require.NoError(t, err)

		rowAffected, err := result.RowsAffected()
		require.NoError(t, err)
		require.Equal(t, int64(1), rowAffected)
	})

	// Test successfully executing a non-query SQL statement in a transaction
	t.Run("transaction", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		err := dbManager.DataDb.RunInTransaction(ctx, func(txCtx context.Context) error {
			_, err := dbManager.DataDb.ExecContext(txCtx, "INSERT INTO test (name) VALUES ('test')")
			if err != nil {
				return err
			}

			return nil
		})

		require.NoError(t, err)

		var count int
		err = dbManager.DataDb.QueryRowContext(ctx, "SELECT COUNT(*) FROM test").Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqliteDb_GetContext(t *testing.T) {
	// Test successfully getting a single record
	t.Run("simple", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		_, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test1'), ('test2')")
		require.NoError(t, err)

		record := &struct {
			Id   int    `db:"id"`
			Name string `db:"name"`
		}{}

		err = dbManager.DataDb.GetContext(ctx, record, "SELECT * FROM test WHERE name = ?", "test1")
		require.NoError(t, err)
		require.Equal(t, 1, record.Id)
		require.Equal(t, "test1", record.Name)
	})

	// Test successfully getting a single record in a transaction
	t.Run("transaction", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		_, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test1'), ('test2')")
		require.NoError(t, err)

		record := &struct {
			Id   int    `db:"id"`
			Name string `db:"name"`
		}{}

		err = dbManager.DataDb.RunInTransaction(ctx, func(txCtx context.Context) error {
			return dbManager.DataDb.GetContext(txCtx, record, "SELECT * FROM test WHERE name = ?", "test1")
		})

		require.NoError(t, err)
		require.Equal(t, 1, record.Id)
		require.Equal(t, "test1", record.Name)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqliteDb_SelectContext(t *testing.T) {
	// Test successfully selecting multiple records
	t.Run("simple", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		records := []*struct {
			Id   int    `db:"id"`
			Name string `db:"name"`
		}{}

		_, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test'), ('test2')")
		require.NoError(t, err)

		err = dbManager.DataDb.SelectContext(ctx, &records, "SELECT * FROM test")
		require.NoError(t, err)
		require.Len(t, records, 2)
		require.Equal(t, 1, records[0].Id)
		require.Equal(t, "test", records[0].Name)
		require.Equal(t, 2, records[1].Id)
		require.Equal(t, "test2", records[1].Name)
	})

	// Test successfully selecting multiple records in a transaction
	t.Run("transaction", func(t *testing.T) {
		dbManager := setupSqliteDB(t)
		ctx := context.Background()

		_, err := dbManager.DataDb.ExecContext(ctx, "INSERT INTO test (name) VALUES ('test'), ('test2')")
		require.NoError(t, err)

		records := []*struct {
			Id   int    `db:"id"`
			Name string `db:"name"`
		}{}

		err = dbManager.DataDb.RunInTransaction(ctx, func(txCtx context.Context) error {
			return dbManager.DataDb.SelectContext(txCtx, &records, "SELECT * FROM test")
		})

		require.NoError(t, err)
		require.Len(t, records, 2)
		require.Equal(t, 1, records[0].Id)
		require.Equal(t, "test", records[0].Name)
		require.Equal(t, 2, records[1].Id)
		require.Equal(t, "test2", records[1].Name)
	})
}
