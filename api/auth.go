package api

import (
	"context"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/auth"
	"github.com/geerew/off-course/utils/types"
	"github.com/gofiber/fiber/v2"
)

// TODO Add unit tests for the auth routes

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type authAPI struct {
	r *Router
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initFsRoutes initializes the filesystem routes
func (r *Router) initAuthRoutes() {
	authAPI := authAPI{r: r}

	authGroup := r.apiGroup("auth")

	authGroup.Get("/signup-status", authAPI.signupStatus)
	authGroup.Post("/bootstrap/:token", authAPI.bootstrap)
	authGroup.Post("/register", authAPI.register)
	authGroup.Post("/login", authAPI.login)
	authGroup.Post("/logout", authAPI.logout)

	authGroup.Get("/me", authAPI.getMe)
	authGroup.Put("/me", authAPI.updateMe)
	authGroup.Delete("/me", authAPI.deleteMe)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api authAPI) signupStatus(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(signupStatusResponse{
		Enabled: api.r.app.Config.EnableSignup,
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api authAPI) bootstrap(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return errorResponse(c, fiber.StatusBadRequest, "Bootstrap token is required", nil)
	}

	// Check if already bootstrapped first
	if api.r.app.IsBootstrapped() {
		return errorResponse(c, fiber.StatusForbidden, "Application is already bootstrapped", nil)
	}

	// Validate bootstrap token
	if err := auth.ValidateBootstrapToken(token, api.r.app.Config.DataDir, api.r.app.FS); err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Invalid or expired bootstrap token", nil)
	}

	// Create admin user using existing register logic
	err := api.register(c)
	if err == nil {
		api.r.app.SetBootstrapped()
		auth.DeleteBootstrapToken(api.r.app.Config.DataDir, api.r.app.FS)
	}

	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api authAPI) register(c *fiber.Ctx) error {
	if api.r.app.IsBootstrapped() && !api.r.app.Config.EnableSignup {
		return errorResponse(c, fiber.StatusForbidden, "Sign-up is disabled", nil)
	}

	registerReq := &registerRequest{}

	if err := c.BodyParser(registerReq); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	if registerReq.Username == "" || registerReq.Password == "" {
		return errorResponse(c, fiber.StatusBadRequest, "Username and/or password cannot be empty", nil)
	}

	if err := validatePassword(registerReq.Password); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	passwordHash, err := auth.GeneratePassword(registerReq.Password)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error hashing password", err)
	}

	user := &models.User{
		Username:     registerReq.Username,
		DisplayName:  registerReq.Username, // Set the display name to the username by default
		PasswordHash: passwordHash,
	}

	// The first user will always be an admin
	if !api.r.app.IsBootstrapped() {
		user.Role = types.UserRoleAdmin
	} else {
		user.Role = types.UserRoleUser
	}

	err = api.r.appDao.CreateUser(c.UserContext(), user)
	if err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return errorResponse(c, fiber.StatusBadRequest, "Username already exists", nil)
		}

		return errorResponse(c, fiber.StatusInternalServerError, "Error creating user", err)
	}

	err = api.r.sessionManager.SetSession(c, user.ID, user.Role)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error setting session", err)
	}
	return c.SendStatus(fiber.StatusCreated)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api authAPI) login(c *fiber.Ctx) error {
	loginReq := &loginRequest{}

	if err := c.BodyParser(loginReq); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	if loginReq.Username == "" || loginReq.Password == "" {
		return errorResponse(c, fiber.StatusBadRequest, "Username and/or password cannot be empty", nil)
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: loginReq.Username})
	user, err := api.r.appDao.GetUser(c.UserContext(), dbOpts)
	if err != nil || user == nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Invalid username and/or password", nil)
	}

	if !auth.ComparePassword(user.PasswordHash, loginReq.Password) {
		return errorResponse(c, fiber.StatusUnauthorized, "Invalid username and/or password", nil)
	}

	err = api.r.sessionManager.SetSession(c, user.ID, user.Role)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error setting session", err)
	}
	return c.SendStatus(fiber.StatusOK)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api authAPI) logout(c *fiber.Ctx) error {
	err := api.r.sessionManager.DeleteSession(c)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting session", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api authAPI) getMe(c *fiber.Ctx) error {
	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	user, err := api.getUserByPrincipal(ctx, principal)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error getting user information", err)
	}

	return c.Status(fiber.StatusOK).JSON(&userResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api authAPI) updateMe(c *fiber.Ctx) error {
	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	user, err := api.getUserByPrincipal(ctx, principal)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error getting user information", err)
	}

	updateReq := &selfUpdateRequest{}
	if err := c.BodyParser(updateReq); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	if updateReq.DisplayName == "" && updateReq.Password == "" {
		return errorResponse(c, fiber.StatusBadRequest, "No data to update", nil)
	}

	if updateReq.DisplayName != "" {
		user.DisplayName = updateReq.DisplayName
	}

	if updateReq.Password != "" {
		if !auth.ComparePassword(user.PasswordHash, updateReq.CurrentPassword) {
			return errorResponse(c, fiber.StatusBadRequest, "Invalid current password", nil)
		}

		passwordHash, err := auth.GeneratePassword(updateReq.Password)
		if err != nil {
			return errorResponse(c, fiber.StatusInternalServerError, "Error hashing password", err)
		}
		user.PasswordHash = passwordHash
	}

	err = api.r.appDao.UpdateUser(ctx, user)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error updating user", err)
	}

	return c.Status(fiber.StatusOK).JSON(&userResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api authAPI) deleteMe(c *fiber.Ctx) error {
	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	user, err := api.getUserByPrincipal(ctx, principal)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error getting user information", err)
	}

	deleteReq := &selfDeleteRequest{}
	if err := c.BodyParser(deleteReq); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	if !auth.ComparePassword(user.PasswordHash, deleteReq.CurrentPassword) {
		return errorResponse(c, fiber.StatusBadRequest, "Invalid password", nil)
	}

	if user.Role == types.UserRoleAdmin {
		// Count the number of admin users and fail if there is only one
		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ROLE: types.UserRoleAdmin})
		adminCount, err := api.r.appDao.CountUsers(ctx, dbOpts)
		if err != nil {
			return errorResponse(c, fiber.StatusInternalServerError, "Error counting admin users", err)
		}

		if adminCount == 1 {
			return errorResponse(c, fiber.StatusBadRequest, "Unable to delete the last admin user", nil)
		}
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: principal.UserID})
	err = api.r.appDao.DeleteUsers(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting user", err)
	}

	err = api.r.sessionManager.DeleteUserSessions(user.ID)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting user sessions", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getUserByPrincipal retrieves a user by the principal's user ID
func (api authAPI) getUserByPrincipal(ctx context.Context, principal types.Principal) (*models.User, error) {
	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: principal.UserID})
	return api.r.appDao.GetUser(ctx, dbOpts)
}
