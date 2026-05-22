package database

import (
	"testing"

	"github.com/geerew/off-course/utils/filesystem"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_NewSQLiteManager(t *testing.T) {
	// Test successfully creating a new SQLiteManager
	t.Run("success", func(t *testing.T) {
		fs := filesystem.New(afero.NewMemMapFs())

		dbManager, err := NewSQLiteManager(&DatabaseManagerConfig{
			DataDir: "./oc_data",
			FS:   fs,
			Testing: true,
		})

		require.NoError(t, err)
		require.NotNil(t, dbManager)

	})

	// Test error due to being unable to create data.db
	t.Run("error creating data.db", func(t *testing.T) {
		fs := filesystem.New(afero.NewReadOnlyFs(afero.NewMemMapFs()))

		dbManager, err := NewSQLiteManager(&DatabaseManagerConfig{
			DataDir: "./oc_data",
			FS:   fs,
			Testing: true,
		})

		require.NotNil(t, err)
		require.Contains(t, err.Error(), "failed to create write database")
		require.Contains(t, err.Error(), "operation not permitted")
		require.Nil(t, dbManager)
	})
}
