package api

import (
	"errors"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/auth"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
	"github.com/gofiber/fiber/v2"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type userAPI struct {
	r *Router
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initFsRoutes initializes the filesystem routes
func (r *Router) initUserRoutes() {
	userAPI := userAPI{
		r: r,
	}

	g := r.apiGroup("users")

	g.Get("", protectedRoute, userAPI.getUsers)
	g.Post("", protectedRoute, userAPI.createUser)
	g.Put("/:id", protectedRoute, userAPI.updateUser)
	g.Delete("/:id", protectedRoute, userAPI.deleteUser)

	g.Delete("/:id/sessions", protectedRoute, userAPI.deleteUserSession)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api userAPI) getUsers(c *fiber.Ctx) error {
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithOrderBy(defaultUsersOrderBy...).
		WithPagination(pagination.NewFromApi(c)).
		WithStringQuery(&dao.StringQuery{
			Query:          c.Query("q", ""),
			AllowedFilters: []string{"role"},
		})

	users, err := api.r.appDao.ListUsers(ctx, dbOpts)
	if err != nil {
		if errors.Is(err, dao.ErrStringQueryParse) {
			return errorResponse(c, fiber.StatusBadRequest, "Error parsing query", err)
		}

		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up users", err)
	}

	pResult, err := dbOpts.Pagination.BuildResult(userResponseHelper(users))
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error building pagination result", err)
	}

	return c.Status(fiber.StatusOK).JSON(pResult)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api userAPI) createUser(c *fiber.Ctx) error {
	userReq := &userRequest{}

	if err := c.BodyParser(userReq); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	if userReq.Username == "" || userReq.Password == "" {
		return errorResponse(c, fiber.StatusBadRequest, "A username and password are required", nil)
	}

	if err := validatePassword(userReq.Password); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, err.Error(), nil)
	}

	// Default the role to a user when not provided
	if userReq.Role == "" {
		userReq.Role = types.UserRoleUser.String()
	}

	user := &models.User{
		Username:     userReq.Username,
		DisplayName:  userReq.Username,
		PasswordHash: auth.GeneratePassword(userReq.Password),
		Role:         types.NewUserRole(userReq.Role),
	}

	if userReq.DisplayName != "" {
		user.DisplayName = userReq.DisplayName
	}

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	if err := api.r.appDao.CreateUser(ctx, user); err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return errorResponse(c, fiber.StatusBadRequest, "Username already exists", nil)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error creating user", err)
	}
	return c.SendStatus(fiber.StatusCreated)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Revokes all sessions for the user when the role is updated
func (api userAPI) updateUser(c *fiber.Ctx) error {
	id := c.Params("id")

	userReq := &userRequest{}
	if err := c.BodyParser(userReq); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	if userReq.DisplayName == "" && userReq.Password == "" && userReq.Role == "" {
		return errorResponse(c, fiber.StatusBadRequest, "No data to update", nil)
	}

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: id})
	user, err := api.r.appDao.GetUser(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up user", err)
	}

	if user == nil {
		return errorResponse(c, fiber.StatusNotFound, "User not found", nil)
	}

	if userReq.DisplayName != "" {
		user.DisplayName = userReq.DisplayName
	}

	if userReq.Password != "" {
		if err := validatePassword(userReq.Password); err != nil {
			return errorResponse(c, fiber.StatusBadRequest, err.Error(), nil)
		}
		user.PasswordHash = auth.GeneratePassword(userReq.Password)
	}

	if userReq.Role != "" {
		// Do nothing when the role is the same
		if user.Role.String() == userReq.Role {
			userReq.Role = ""
		} else {
			user.Role = types.NewUserRole(userReq.Role)
		}
	}

	err = api.r.appDao.UpdateUser(ctx, user)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error updating user", err)
	}

	// Update all the sessions for the user with the new role
	if userReq.Role != "" {
		if err := api.r.sessionManager.UpdateSessionRoleForUser(id, user.Role); err != nil {
			return errorResponse(c, fiber.StatusInternalServerError, "Error updating user sessions", err)
		}
	}

	return c.Status(fiber.StatusOK).JSON(&userResponse{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Role:        user.Role,
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api userAPI) deleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: id})
	err = api.r.appDao.DeleteUsers(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting user", err)
	}

	err = api.r.sessionManager.DeleteUserSessions(id)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting user sessions", err)
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api userAPI) deleteUserSession(c *fiber.Ctx) error {
	id := c.Params("id")

	err := api.r.sessionManager.DeleteUserSessions(id)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting user sessions", err)
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
