package dao

import (
	"context"
	"testing"

	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func setup(tb testing.TB) (*DAO, context.Context) {
	tb.Helper()

	// DB
	dbManager, err := database.NewSQLiteManager(&database.DatabaseManagerConfig{
		DataDir: "./oc_data",
		AppFs:   appfs.New(afero.NewMemMapFs()),
		Testing: true,
	})

	require.NoError(tb, err)
	require.NotNil(tb, dbManager)

	dao := &DAO{db: dbManager.DataDb}

	// User
	user := &models.User{
		Username:     "test-user",
		DisplayName:  "Test User",
		PasswordHash: "test-password",
		Role:         types.UserRoleAdmin,
	}
	require.NoError(tb, dao.CreateUser(context.Background(), user))

	ctx := context.Background()
	principal := types.Principal{
		UserID: user.ID,
		Role:   user.Role,
	}
	ctx = context.WithValue(ctx, types.PrincipalContextKey, principal)

	return dao, ctx
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func setupLog(tb testing.TB) (*DAO, context.Context) {
	tb.Helper()

	appFs := appfs.New(afero.NewMemMapFs())

	dbManager, err := database.NewSQLiteManager(&database.DatabaseManagerConfig{
		DataDir: "./oc_data",
		AppFs:   appFs,
		Testing: true,
	})

	require.NoError(tb, err)
	require.NotNil(tb, dbManager)

	dao := &DAO{db: dbManager.LogsDb}

	return dao, context.Background()
}
