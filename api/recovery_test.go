package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/auth"
	"github.com/geerew/off-course/utils/security"
	"github.com/geerew/off-course/utils/types"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestRecovery_ResetPassword(t *testing.T) {
	t.Run("200 (success)", func(t *testing.T) {
		router, ctx := setupNoAuth(t)

		// Create admin user
		user := &models.User{
			Username:     "testadmin",
			DisplayName:  "Test Admin",
			PasswordHash: security.RandomString(10),
			Role:         types.UserRoleAdmin,
		}
		require.NoError(t, router.appDao.CreateUser(ctx, user))

		// Generate recovery token in the router's data directory
		recoveryToken, err := auth.GenerateRecoveryToken(router.app.FS, "testadmin", "newpassword123", router.app.Config.DataDir)
		require.NoError(t, err)

		// Make request
		reqBody := map[string]string{
			"token": recoveryToken.Token,
		}
		jsonData, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/recovery", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		// Verify response
		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		require.Equal(t, "Password reset successfully", response["message"])
		require.Equal(t, "testadmin", response["username"])

		// Verify password was updated
		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "testadmin"})
		updatedUser, err := router.appDao.GetUser(ctx, dbOpts)
		require.NoError(t, err)
		require.True(t, auth.ComparePassword(updatedUser.PasswordHash, "newpassword123"))
	})

	t.Run("401 (invalid token)", func(t *testing.T) {
		router, _ := setupNoAuth(t)

		// Make request with invalid token
		reqBody := map[string]string{
			"token": "invalid-token",
		}
		jsonData, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/recovery", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, status)
	})

	t.Run("401 (no token file)", func(t *testing.T) {
		router, _ := setupNoAuth(t)

		// Make request with token but no file
		reqBody := map[string]string{
			"token": "some-token",
		}
		jsonData, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/recovery", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, status)
	})

	t.Run("400 (missing token)", func(t *testing.T) {
		router, _ := setupNoAuth(t)

		// Make request without token
		req := httptest.NewRequest(http.MethodPost, "/api/admin/recovery", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
	})

	t.Run("404 (user not found)", func(t *testing.T) {
		router, _ := setupNoAuth(t)

		// Generate recovery token for non-existent user
		recoveryToken, err := auth.GenerateRecoveryToken(router.app.FS, "nonexistent", "password", router.app.Config.DataDir)
		require.NoError(t, err)

		// Make request
		reqBody := map[string]string{
			"token": recoveryToken.Token,
		}
		jsonData, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/recovery", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
	})

	t.Run("403 (user not admin)", func(t *testing.T) {
		ctx := context.Background()
		router, _ := setupNoAuth(t)

		// Create regular user (not admin)
		user := &models.User{
			Username:     "testuser",
			DisplayName:  "Test User",
			PasswordHash: security.RandomString(10),
			Role:         types.UserRoleUser,
		}
		require.NoError(t, router.appDao.CreateUser(ctx, user))

		// Generate recovery token
		recoveryToken, err := auth.GenerateRecoveryToken(router.app.FS, "testuser", "newpassword123", router.app.Config.DataDir)
		require.NoError(t, err)

		// Make request
		reqBody := map[string]string{
			"token": recoveryToken.Token,
		}
		jsonData, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/api/admin/recovery", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusForbidden, status)
	})
}
