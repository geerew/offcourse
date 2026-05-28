package api

import (
	"errors"
	"net/url"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/coursemetadata"
	"github.com/gofiber/fiber/v2"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type tagsAPI struct {
	r *Router
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initTagRoutes initializes the tag routes
func (r *Router) initTagRoutes() {
	tagsAPI := tagsAPI{
		r: r,
	}

	g := r.apiGroup("tags")

	g.Get("", tagsAPI.getTags)
	g.Get("/names", tagsAPI.getTagNames)
	g.Get("/:name", tagsAPI.getTag)
	g.Post("", protectedRoute, tagsAPI.createTag)
	g.Put("/:id", protectedRoute, tagsAPI.updateTag)
	g.Delete("/:id", protectedRoute, tagsAPI.deleteTag)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api *tagsAPI) getTags(c *fiber.Ctx) error {
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithOrderBy(utils.StringSplit(c.Query("orderBy", ""), ",")...).
		WithApiQuery(c.Query("q", "")).
		WithPagination(paginationFromCtx(c))

	tags, err := api.r.appDao.ListTags(ctx, dbOpts)
	if err != nil {
		if errors.Is(err, utils.ErrApiQueryParse) {
			return errorResponse(c, fiber.StatusBadRequest, "Error parsing query", err)
		}

		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up tags", err)
	}

	pResult, err := dbOpts.Pagination.BuildResult(tagResponseHelper(tags))
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error building pagination result", err)
	}

	return c.Status(fiber.StatusOK).JSON(pResult)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api *tagsAPI) getTagNames(c *fiber.Ctx) error {
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithOrderBy(utils.StringSplit(c.Query("orderBy", ""), ",")...).
		WithApiQuery(c.Query("q", ""))

	tags, err := api.r.appDao.ListTagNames(ctx, dbOpts)
	if err != nil {
		if errors.Is(err, utils.ErrApiQueryParse) {
			return errorResponse(c, fiber.StatusBadRequest, "Error parsing query", err)
		}

		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up tags", err)
	}

	return c.Status(fiber.StatusOK).JSON(tags)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api *tagsAPI) getTag(c *fiber.Ctx) error {
	name := c.Params("name")

	var err error
	name, err = url.QueryUnescape(name)

	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error decoding name parameter", err)
	}

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_TAG: name})

	tag, err := api.r.appDao.GetTag(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up tag", err)
	}

	if tag == nil {
		return errorResponse(c, fiber.StatusNotFound, "Tag not found", nil)
	}

	return c.Status(fiber.StatusOK).JSON(tagResponseHelper([]*models.Tag{tag})[0])
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api *tagsAPI) createTag(c *fiber.Ctx) error {
	req := &tagRequest{}
	if err := c.BodyParser(req); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	if req.Tag == "" {
		return errorResponse(c, fiber.StatusBadRequest, "A tag is required", nil)
	}

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	tag := &models.Tag{Tag: req.Tag}
	if err := api.r.appDao.CreateTag(ctx, tag); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return errorResponse(c, fiber.StatusBadRequest, "Tag already exists", err)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error creating tag", err)
	}

	return c.Status(fiber.StatusCreated).JSON(tagResponseHelper([]*models.Tag{tag})[0])
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api *tagsAPI) updateTag(c *fiber.Ctx) error {
	id := c.Params("id")

	tagReq := &tagRequest{}
	if err := c.BodyParser(tagReq); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: id})
	tag, err := api.r.appDao.GetTag(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up tag", err)
	}

	if tag == nil {
		return errorResponse(c, fiber.StatusNotFound, "Tag not found", nil)
	}

	tag.Tag = tagReq.Tag

	if err := api.r.appDao.UpdateTag(ctx, tag); err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return errorResponse(c, fiber.StatusBadRequest, "Tag already exists", err)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error updating tag", err)
	}

	return c.Status(fiber.StatusOK).JSON(tagResponseHelper([]*models.Tag{tag})[0])
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api *tagsAPI) deleteTag(c *fiber.Ctx) error {
	id := c.Params("id")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Get the tag before deleting it to know which courses are affected
	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: id})
	tag, err := api.r.appDao.GetTag(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up tag", err)
	}

	// If tag doesn't exist, return 204 (idempotent delete)
	if tag == nil {
		return c.Status(fiber.StatusNoContent).Send(nil)
	}

	// Find all courses that have this tag (before deletion)
	dbOpts = dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_TAG_ID: id})
	courseTags, err := api.r.appDao.ListCourseTags(ctx, dbOpts)
	if err != nil {
		// If reading fails, still proceed with deletion but skip file updates
		courseTags = []*models.CourseTag{}
	}

	// Collect unique course IDs and get their paths
	courseMap := make(map[string]string) // courseID -> path
	for _, ct := range courseTags {
		if _, exists := courseMap[ct.CourseID]; !exists {
			// Get course to access its path
			course, err := api.r.appDao.GetCourse(ctx, dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: ct.CourseID}))
			if err != nil || course == nil {
				continue // Skip if course not found
			}
			courseMap[ct.CourseID] = course.Path
		}
	}

	// Delete the tag (this will cascade delete course_tags)
	dbOpts = dao.NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: id})
	if err = api.r.appDao.DeleteTags(ctx, dbOpts); err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting tag", err)
	}

	// Now read the actual remaining tags AFTER deletion to avoid race conditions
	// This ensures we get the correct state even if other tags were deleted concurrently
	for courseID, coursePath := range courseMap {
		// Get all remaining tags for this course (after deletion)
		courseDbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courseID})
		remainingCourseTags, err := api.r.appDao.ListCourseTags(ctx, courseDbOpts)
		if err != nil {
			continue // Skip if can't get tags
		}

		// Build list of remaining tag names
		remainingTags := make([]string, 0, len(remainingCourseTags))
		for _, t := range remainingCourseTags {
			remainingTags = append(remainingTags, t.Tag)
		}

		// Update oc.json file with actual remaining tags
		metadata := &coursemetadata.CourseMetadata{
			Description: "",
			Tags:        remainingTags,
		}
		api.r.app.MetadataWriter.WriteMetadataAsync(courseID, coursePath, metadata)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
