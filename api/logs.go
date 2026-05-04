package api

import (
	"errors"

	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/gofiber/fiber/v2"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type logsAPI struct {
	r *Router
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initLogRoutes initializes the log routes
func (r *Router) initLogRoutes() {
	logsAPI := logsAPI{
		r: r,
	}

	g := r.apiGroup("logs")
	g.Get("/", protectedRoute, logsAPI.getLogs)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api *logsAPI) getLogs(c *fiber.Ctx) error {
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithPagination(pagination.NewFromApi(c)).
		WithApiQuery(c.Query("q", ""))

	logs, err := api.r.logDao.ListLogs(ctx, dbOpts)
	if err != nil {
		if errors.Is(err, utils.ErrApiQueryParse) {
			return errorResponse(c, fiber.StatusBadRequest, "Error parsing query", err)
		}

		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up logs", err)
	}

	pResult, err := dbOpts.Pagination.BuildResult(logsResponseHelper(logs))
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error building pagination result", err)
	}

	return c.Status(fiber.StatusOK).JSON(pResult)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
