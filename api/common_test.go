package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/geerew/off-course/app"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setupAdmin creates a test router with an admin user
func setupAdmin(t *testing.T) (*Router, context.Context) {
	return setup(t, "admin", types.UserRoleAdmin)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setupUser creates a test router with a regular user
func setupUser(t *testing.T) (*Router, context.Context) {
	return setup(t, "user", types.UserRoleUser)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setupNoAuth creates a test router without authentication
//
// Note: role doesn't matter when the user is empty
func setupNoAuth(t *testing.T) (*Router, context.Context) {
	return setup(t, "", types.UserRoleUser)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setup creates a test router
func setup(t *testing.T, id string, role types.UserRole) (*Router, context.Context) {
	t.Helper()

	// Create test app
	application := app.NewTestApp(t)

	// Create router from app
	router := NewRouter(application)

	// Configure middleware based on whether we have auth
	if id != "" {
		// Use dev auth for normal tests
		router.SetTestMiddleware(
			func(r *Router) fiber.Handler { return requestLoggingMiddleware(r.logger) },
			func(r *Router) fiber.Handler { return corsMiddleWare() },
			func(r *Router) fiber.Handler { return bootstrapMiddleware(r) },
			func(r *Router) fiber.Handler {
				return func(c *fiber.Ctx) error {
					c.Locals(types.PrincipalContextKey, types.Principal{
						UserID: id,
						Role:   role,
					})
					return c.Next()
				}
			},
		)
	} else {
		// Use CORS-only (for recovery tests)
		router.SetTestMiddleware(
			func(r *Router) fiber.Handler { return corsMiddleWare() },
		)
	}

	// Create user only if we have auth
	if id != "" {
		user := models.User{
			Base: models.Base{
				ID: id,
			},
			Username:     id,
			Role:         role,
			PasswordHash: "password",
			DisplayName:  "Test User",
		}

		require.NoError(t, router.appDao.CreateUser(context.Background(), &user))

		if role == types.UserRoleAdmin {
			router.app.IsBootstrapped()
		}
	}

	// In tests, if we have a user, consider the app bootstrapped
	// (even if the user is not an admin, so protectedRoute can handle the check)
	if id != "" {
		router.app.SetBootstrapped()
	}

	ctx := context.Background()

	// Add principal to context only if we have auth
	if id != "" {
		principal := types.Principal{
			UserID: id,
			Role:   role,
		}
		ctx = context.WithValue(ctx, types.PrincipalContextKey, principal)
	}

	return router, ctx
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// requestHelper sends a request to the router and returns the status code and body
func requestHelper(t *testing.T, router *Router, req *http.Request) (int, []byte, error) {
	t.Helper()

	resp, err := router.Test(req)
	if err != nil {
		return -1, nil, err
	}

	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// unmarshalHelper unmarshals a body into a pagination result and a slice of items
func unmarshalHelper[T any](t *testing.T, body []byte) (pagination.PaginationResult, []T) {
	t.Helper()

	var respData pagination.PaginationResult
	err := json.Unmarshal(body, &respData)
	require.NoError(t, err)

	var resp []T
	for _, item := range respData.Items {
		var r T
		require.Nil(t, json.Unmarshal(item, &r))
		resp = append(resp, r)
	}

	return respData, resp
}
