package cron

import (
	"context"
	"testing"

	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/filesystem"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// testApp holds the dependencies needed for cron tests
type testApp struct {
	DbManager *database.DatabaseManager
	FS        *filesystem.FS
	Logger    *logger.Logger
}

func setup(t *testing.T) (*testApp, context.Context) {
	t.Helper()

	// Create filesystem
	fs := filesystem.New(afero.NewMemMapFs())

	// Create database manager
	dbManagerConfig := &database.DatabaseManagerConfig{
		DataDir: "./oc_data",
		FS:   fs,
		Testing: true,
	}

	dbManager, err := database.NewSQLiteManager(dbManagerConfig)
	require.NoError(t, err)

	// Create logger
	appLogger := logger.NilLogger()

	// Create DAO to create a user
	appDao := dao.New(dbManager.DataDb)

	// User
	user := &models.User{
		Username:     "test-user",
		DisplayName:  "Test User",
		PasswordHash: "test-password",
		Role:         types.UserRoleAdmin,
	}
	require.NoError(t, appDao.CreateUser(context.Background(), user))

	principal := types.Principal{
		UserID: user.ID,
		Role:   user.Role,
	}

	ctx := context.WithValue(context.Background(), types.PrincipalContextKey, principal)

	return &testApp{
		DbManager: dbManager,
		FS:   fs,
		Logger:    appLogger,
	}, ctx
}
