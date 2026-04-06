package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/coursemetadata"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type coursesAPI struct {
	r *Router
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initCourseRoutes initializes the course routes
func (r *Router) initCourseRoutes() {
	coursesAPI := coursesAPI{
		r: r,
	}

	g := r.apiGroup("courses")

	// Course
	g.Get("", coursesAPI.getCourses)
	g.Get("/:id", coursesAPI.getCourse)
	g.Post("", protectedRoute, coursesAPI.createCourse)
	g.Delete("/:id", protectedRoute, coursesAPI.deleteCourse)

	// Progress
	g.Delete("/:id/progress", coursesAPI.deleteCourseProgress)

	// Card
	g.Head("/:id/card", coursesAPI.getCourseCard)
	g.Get("/:id/card", coursesAPI.getCourseCard)

	// Lessons
	g.Get("/:id/lessons", coursesAPI.getCourseLessons)
	g.Get("/:id/lessons/:lesson", coursesAPI.getCourseLesson)

	// Modules (chaptered lessons)
	g.Get("/:id/modules", coursesAPI.getCourseModules)

	// lesson attachments
	g.Get("/:id/lessons/:lesson/attachments", coursesAPI.getCourseLessonAttachments)
	g.Get("/:id/lessons/:lesson/attachments/:attachment", coursesAPI.getCourseLessonAttachment)
	g.Get("/:id/lessons/:lesson/attachments/:attachment/serve", coursesAPI.serveCourseLessonAttachment)

	// Asset
	g.Get("/:id/lessons/:lesson/assets/:asset/serve", coursesAPI.serveCourseLessonAsset)
	g.Put("/:id/lessons/:lesson/assets/:asset/progress", coursesAPI.updateCourseLessonAssetProgress)
	g.Delete("/:id/lessons/:lesson/assets/:asset/progress", coursesAPI.deleteCourseLessonAssetProgress)

	// Tags
	g.Get("/:id/tags", coursesAPI.getCourseTags)
	g.Post("/:id/tags", protectedRoute, coursesAPI.createCourseTag)
	g.Delete("/:id/tags/:tagId", protectedRoute, coursesAPI.deleteCourseTag)

	// Favourites
	g.Post("/:id/favourite", protectedRoute, coursesAPI.favouriteCourse)
	g.Delete("/:id/favourite", protectedRoute, coursesAPI.unfavouriteCourse)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TODO add tests when initial scan is false
func (api coursesAPI) getCourses(c *fiber.Ctx) error {
	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	allowedQueryFilters := []string{"available", "tag"}

	withUserProgress := false
	if raw := c.Query("withUserProgress"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil && v {
			withUserProgress = v
		}
	}

	if withUserProgress {
		allowedQueryFilters = append(allowedQueryFilters, "progress", "favourite")
	}

	dbOpts := dao.NewOptions().
		WithOrderBy(defaultCoursesOrderBy...).
		WithPagination(pagination.NewFromApi(c)).
		WithStringQuery(&dao.StringQuery{
			Query:          c.Query("q", ""),
			AllowedFilters: allowedQueryFilters,
		})

	if withUserProgress {
		dbOpts.WithUserProgress()
	}

	courses, err := api.r.appDao.ListCourses(ctx, dbOpts)
	if err != nil {
		if errors.Is(err, dao.ErrStringQueryParse) {
			return errorResponse(c, fiber.StatusBadRequest, "Error parsing query", err)
		}

		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up courses", err)
	}

	pResult, err := dbOpts.Pagination.BuildResult(courseResponseHelper(courses, principal.Role == types.UserRoleAdmin))
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error building pagination result", err)
	}

	return c.Status(fiber.StatusOK).JSON(pResult)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TODO add tests when initial scan is false
func (api coursesAPI) getCourse(c *fiber.Ctx) error {
	id := c.Params("id")

	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	var courseOpts []func(*dao.Options)
	if raw := c.Query("withUserProgress"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil && v {
			courseOpts = append(courseOpts, func(opts *dao.Options) {
				opts.WithUserProgress()
			})
		}
	}

	course, err := api.getCourseByID(ctx, id, courseOpts...)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up course", err)
	}

	if course == nil {
		return errorResponse(c, fiber.StatusNotFound, "Course not found", fmt.Errorf("course not found"))
	}

	return c.Status(fiber.StatusOK).JSON(courseResponseHelper([]*models.Course{course}, principal.Role == types.UserRoleAdmin)[0])
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) createCourse(c *fiber.Ctx) error {
	req := &courseRequest{}
	if err := c.BodyParser(req); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	// Ensure there is a title and path
	if req.Title == "" || req.Path == "" {
		return errorResponse(c, fiber.StatusBadRequest, "A title and path are required", nil)
	}

	course := &models.Course{
		Title: req.Title,
		Path:  utils.NormalizeWindowsDrive(req.Path),
	}

	// Validate the path
	if exists, err := afero.DirExists(api.r.app.AppFs.Fs, course.Path); err != nil || !exists {
		return errorResponse(c, fiber.StatusBadRequest, "Invalid course path", err)
	}

	// Set the course to available
	course.Available = true

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	if err := api.r.appDao.CreateCourse(ctx, course); err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return errorResponse(c, fiber.StatusBadRequest, "A course with this path already exists", err)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error creating course", err)
	}

	// Start a scan job
	if _, err := api.r.app.CourseScan.Add(ctx, course.ID); err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error creating scan job", err)
	}

	return c.Status(fiber.StatusCreated).JSON(courseResponseHelper([]*models.Course{course}, true)[0])
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) deleteCourse(c *fiber.Ctx) error {
	id := c.Params("id")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Cancel and remove any ongoing scans for this course
	api.r.app.CourseScan.CancelAndRemoveScansByCourseID(id)

	// Delete optimized card file if it exists
	cardPath := api.r.app.CardCache.GetCardPath(id)
	if err := api.r.app.CardCache.DeleteCard(cardPath); err != nil {
		// Log warning but continue with course deletion
		api.r.app.Logger.Warn().
			Err(err).
			Str("course_id", id).
			Str("card_path", cardPath).
			Msg("Failed to delete optimized card during course deletion")
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: id})
	if err := api.r.appDao.DeleteCourses(ctx, dbOpts); err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting course", err)
	}
	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TODO add tests
func (api coursesAPI) deleteCourseProgress(c *fiber.Ctx) error {
	courseId := c.Params("id")

	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	err = api.r.appDao.RunInTransaction(ctx, func(txCtx context.Context) error {
		// Delete the course progress for this user
		dbOpts := dao.NewOptions().WithWhere(squirrel.And{
			squirrel.Eq{models.COURSE_PROGRESS_COURSE_ID: courseId},
			squirrel.Eq{models.COURSE_PROGRESS_USER_ID: principal.UserID},
		})

		if err := api.r.appDao.DeleteCourseProgress(txCtx, dbOpts); err != nil {
			return err
		}

		dbOpts = dao.NewOptions().WithWhere(squirrel.And{
			squirrel.Eq{models.ASSET_PROGRESS_TABLE_USER_ID: principal.UserID},
			squirrel.Expr(
				"EXISTS (SELECT 1 FROM "+models.ASSET_TABLE+
					" WHERE "+models.ASSET_TABLE_ID+" = "+models.ASSET_PROGRESS_TABLE_ASSET_ID+
					" AND "+models.ASSET_TABLE_COURSE_ID+" = ?)",
				courseId,
			),
		})

		if err := api.r.appDao.DeleteAssetProgress(txCtx, dbOpts); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting course progress", err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) getCourseCard(c *fiber.Ctx) error {
	id := c.Params("id")

	_, _, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Check if optimized card exists for this course
	cardPath := api.r.app.CardCache.GetCardPath(id)
	exists, err := api.r.app.CardCache.CardExists(cardPath)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error checking card", err)
	}

	// If card doesn't exist, serve fallback
	if !exists {
		cardPath = api.r.app.CardCache.GetFallbackPath()
		exists, err := api.r.app.CardCache.CardExists(cardPath)
		if err != nil {
			return errorResponse(c, fiber.StatusInternalServerError, "Error checking fallback card", err)
		}
		if !exists {
			return errorResponse(c, fiber.StatusNotFound, "Fallback card not found", nil)
		}
	}

	c.Set(fiber.HeaderCacheControl, "public, no-cache")

	// The fiber function sendFile(...) does not support using a custom FS. Therefore, use
	// SendFile() from the filesystem middleware
	return filesystem.SendFile(c, afero.NewHttpFs(api.r.app.AppFs.Fs), cardPath)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TODO support chaptered query param
func (api coursesAPI) getCourseLessons(c *fiber.Ctx) error {
	id := c.Params("id")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithOrderBy(defaultCourseLessonsOrderBy...).
		WithPagination(pagination.NewFromApi(c)).
		WithStringQuery(&dao.StringQuery{Query: c.Query("q", "")}).
		WithAssetMetadata().
		WithWhere(squirrel.Eq{models.LESSON_TABLE_COURSE_ID: id})

	if withUserProgress := c.Query("withUserProgress"); withUserProgress != "" {
		if v, err := strconv.ParseBool(withUserProgress); err == nil && v {
			dbOpts.WithUserProgress()
		}
	}

	lessons, err := api.r.appDao.ListLessons(ctx, dbOpts)
	if err != nil {
		if errors.Is(err, dao.ErrStringQueryParse) {
			return errorResponse(c, fiber.StatusBadRequest, "Error parsing query", err)
		}

		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up lessons", err)
	}

	pResult, err := dbOpts.Pagination.BuildResult(lessonResponseHelper(lessons))
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error building pagination result", err)
	}

	return c.Status(fiber.StatusOK).JSON(pResult)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) getCourseLesson(c *fiber.Ctx) error {
	id := c.Params("id")
	lessonId := c.Params("lesson")

	dbOpts := dao.NewOptions().
		WithAssetMetadata().
		WithWhere(squirrel.And{
			squirrel.Eq{models.LESSON_TABLE_ID: lessonId},
			squirrel.Eq{models.LESSON_TABLE_COURSE_ID: id},
		})

	if raw := c.Query("withUserProgress"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil && v {
			dbOpts.WithUserProgress()
		}
	}

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	lesson, err := api.r.appDao.GetLesson(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up lesson", err)
	}

	if lesson == nil {
		return errorResponse(c, fiber.StatusNotFound, "Lesson not found", nil)
	}

	return c.Status(fiber.StatusOK).JSON(lessonResponseHelper([]*models.Lesson{lesson})[0])
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) getCourseModules(c *fiber.Ctx) error {
	id := c.Params("id")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithOrderBy(defaultCourseLessonsOrderBy...).
		WithStringQuery(&dao.StringQuery{Query: c.Query("q", "")}).
		WithAssetMetadata().
		WithWhere(squirrel.Eq{models.LESSON_TABLE_COURSE_ID: id})

	if withUserProgress := c.Query("withUserProgress"); withUserProgress != "" {
		if v, err := strconv.ParseBool(withUserProgress); err == nil && v {
			dbOpts.WithUserProgress()
		}
	}

	lessons, err := api.r.appDao.ListLessons(ctx, dbOpts)
	if err != nil {
		if errors.Is(err, dao.ErrStringQueryParse) {
			return errorResponse(c, fiber.StatusBadRequest, "Error parsing query", err)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up lessons", err)
	}

	return c.Status(fiber.StatusOK).JSON(modulesResponseHelper(lessons))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) getCourseLessonAttachments(c *fiber.Ctx) error {
	id := c.Params("id")
	lessonId := c.Params("lesson")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithOrderBy(defaultCourseLessonAttachmentsOrderBy...).
		WithPagination(pagination.NewFromApi(c)).
		WithStringQuery(&dao.StringQuery{Query: c.Query("q", "")}).
		WithWhere(squirrel.And{
			squirrel.Eq{models.ATTACHMENT_TABLE_LESSON_ID: lessonId},
			squirrel.Eq{models.ATTACHMENT_TABLE_COURSE_ID: id},
		})

	attachments, err := api.r.appDao.ListAttachments(ctx, dbOpts)
	if err != nil {
		if errors.Is(err, dao.ErrStringQueryParse) {
			return errorResponse(c, fiber.StatusBadRequest, "Error parsing query", err)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up attachments", err)
	}

	pResult, err := dbOpts.Pagination.BuildResult(attachmentResponseHelper(attachments))
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error building pagination result", err)
	}

	return c.Status(fiber.StatusOK).JSON(pResult)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) getCourseLessonAttachment(c *fiber.Ctx) error {
	id := c.Params("id")
	lessonId := c.Params("lesson")
	attachmentId := c.Params("attachment")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.ATTACHMENT_TABLE_ID: attachmentId},
			squirrel.Eq{models.ATTACHMENT_TABLE_LESSON_ID: lessonId},
			squirrel.Eq{models.ATTACHMENT_TABLE_COURSE_ID: id},
		})

	attachment, err := api.r.appDao.GetAttachment(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up attachment", err)
	}

	if attachment == nil {
		return errorResponse(c, fiber.StatusNotFound, "Attachment not found", nil)
	}

	return c.Status(fiber.StatusOK).JSON(attachmentResponseHelper([]*models.Attachment{attachment})[0])
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) serveCourseLessonAttachment(c *fiber.Ctx) error {
	id := c.Params("id")
	lessonId := c.Params("lesson")
	attachmentId := c.Params("attachment")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.ATTACHMENT_TABLE_ID: attachmentId},
			squirrel.Eq{models.ATTACHMENT_TABLE_LESSON_ID: lessonId},
			squirrel.Eq{models.ATTACHMENT_TABLE_COURSE_ID: id},
		})

	attachment, err := api.r.appDao.GetAttachment(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up attachment", err)
	}

	if attachment == nil {
		return errorResponse(c, fiber.StatusNotFound, "Attachment not found", nil)
	}

	if exists, err := afero.Exists(api.r.app.AppFs.Fs, attachment.Path); err != nil || !exists {
		return errorResponse(c, fiber.StatusBadRequest, "Attachment does not exist", err)
	}

	c.Set(fiber.HeaderContentDisposition, `attachment; filename="`+attachment.Title+`"`)
	return filesystem.SendFile(c, afero.NewHttpFs(api.r.app.AppFs.Fs), attachment.Path)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TODO Handle PDF
func (api coursesAPI) serveCourseLessonAsset(c *fiber.Ctx) error {
	id := c.Params("id")
	lessonId := c.Params("lesson")
	assetId := c.Params("asset")

	dbOpts := dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.ASSET_TABLE_COURSE_ID: id},
			squirrel.Eq{models.ASSET_TABLE_LESSON_ID: lessonId},
			squirrel.Eq{models.ASSET_TABLE_ID: assetId},
		})

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	asset, err := api.r.appDao.GetAsset(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up asset", err)
	}

	if asset == nil {
		return errorResponse(c, fiber.StatusNotFound, "Asset not found", nil)
	}

	// Check for invalid path
	if exists, err := afero.Exists(api.r.app.AppFs.Fs, asset.Path); err != nil || !exists {
		return errorResponse(c, fiber.StatusBadRequest, "Asset does not exist", nil)
	}

	if asset.Type.IsVideo() {
		return handleVideo(c, api.r.app.AppFs, asset)
	} else if asset.Type.IsText() || asset.Type.IsMarkdown() {
		return handleText(c, api.r.app.AppFs, asset)
	}

	return c.Status(fiber.StatusOK).SendString("done")
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) updateCourseLessonAssetProgress(c *fiber.Ctx) error {
	courseId := c.Params("id")
	assetId := c.Params("asset")

	req := &assetProgressRequest{}
	if err := c.BodyParser(req); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// First, verify the asset belongs to the specified course
	asset, err := api.r.appDao.GetAsset(ctx, dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.ASSET_TABLE_ID: assetId},
			squirrel.Eq{models.ASSET_TABLE_COURSE_ID: courseId},
		}))

	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up asset", err)
	}

	if asset == nil {
		return errorResponse(c, fiber.StatusNotFound, "Asset not found for this course", nil)
	}

	assetProgress := &models.AssetProgress{
		AssetID:   assetId,
		Position:  req.Position,
		Completed: req.Completed,
	}

	if err := api.r.appDao.UpsertAssetProgress(ctx, assetProgress); err != nil {
		if err == sql.ErrNoRows {
			return errorResponse(c, fiber.StatusNotFound, "Asset not found", nil)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error updating asset progress", err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TODO add tests
func (api coursesAPI) deleteCourseLessonAssetProgress(c *fiber.Ctx) error {
	courseId := c.Params("id")
	assetId := c.Params("asset")

	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// First, verify the asset belongs to the specified course
	asset, err := api.r.appDao.GetAsset(ctx, dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.ASSET_TABLE_ID: assetId},
			squirrel.Eq{models.ASSET_TABLE_COURSE_ID: courseId},
		}))

	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up asset", err)
	}

	if asset == nil {
		return errorResponse(c, fiber.StatusNotFound, "Asset not found for this course", nil)
	}

	dbOpts := dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.ASSET_PROGRESS_ASSET_ID: assetId},
			squirrel.Eq{models.ASSET_PROGRESS_USER_ID: principal.UserID},
		})

	if err := api.r.appDao.DeleteAssetProgress(ctx, dbOpts); err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting asset progress", err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) getCourseTags(c *fiber.Ctx) error {
	id := c.Params("id")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithOrderBy(defaultTagsOrderBy...).
		WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: id})

	courseTags, err := api.r.appDao.ListCourseTags(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up course tags", err)
	}

	return c.Status(fiber.StatusOK).JSON(courseTagResponseHelper(courseTags))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) createCourseTag(c *fiber.Ctx) error {
	courseId := c.Params("id")
	tagRequest := &tagRequest{}

	if err := c.BodyParser(tagRequest); err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Error parsing data", err)
	}

	if tagRequest.Tag == "" {
		return errorResponse(c, fiber.StatusBadRequest, "A tag is required", nil)
	}

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Check if scan is in progress
	if api.r.app.CourseScan.IsScanning(courseId) {
		return errorResponse(c, fiber.StatusConflict, "Cannot modify tags while course scan is in progress", nil)
	}

	// Get course to access its path
	course, err := api.getCourseByID(ctx, courseId)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up course", err)
	}

	if course == nil {
		return errorResponse(c, fiber.StatusNotFound, "Course not found", nil)
	}

	// Check if tag already exists (case-insensitive) before creating
	dbOpts := dao.NewOptions().
		WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courseId})
	existingCourseTags, err := api.r.appDao.ListCourseTags(ctx, dbOpts)
	if err != nil {
		// If reading fails, use empty list - the DB create will handle the error
		existingCourseTags = []*models.CourseTag{}
	}

	// Check if tag already exists (case-insensitive)
	newTagLower := strings.ToLower(tagRequest.Tag)
	for _, ct := range existingCourseTags {
		if strings.ToLower(ct.Tag) == newTagLower {
			return errorResponse(c, fiber.StatusBadRequest, "A tag for this course already exists", nil)
		}
	}

	// Create tag in DB first (synchronous)
	courseTag := &models.CourseTag{
		CourseID: courseId,
		Tag:      tagRequest.Tag,
	}

	if err := api.r.appDao.CreateCourseTag(ctx, courseTag); err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return errorResponse(c, fiber.StatusBadRequest, "A tag for this course already exists", err)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error creating course tag", err)
	}

	// Read actual tags from DB AFTER creation to avoid race conditions with concurrent additions
	// This ensures we get the correct state even if other tags were added concurrently
	dbOpts = dao.NewOptions().
		WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courseId})
	allCourseTags, err := api.r.appDao.ListCourseTags(ctx, dbOpts)
	if err != nil {
		// If reading fails, fall back to building from existing + new tag
		allTags := make([]string, 0, len(existingCourseTags)+1)
		for _, ct := range existingCourseTags {
			allTags = append(allTags, ct.Tag)
		}
		allTags = append(allTags, tagRequest.Tag)
		metadata := &coursemetadata.CourseMetadata{
			Description: "",
			Tags:        allTags,
		}
		api.r.app.MetadataWriter.WriteMetadataAsync(courseId, course.Path, metadata)
	} else {
		// Build list from actual DB state
		allTags := make([]string, 0, len(allCourseTags))
		for _, ct := range allCourseTags {
			allTags = append(allTags, ct.Tag)
		}

		// Queue async file write (fire and forget)
		metadata := &coursemetadata.CourseMetadata{
			Description: "",
			Tags:        allTags,
		}
		api.r.app.MetadataWriter.WriteMetadataAsync(courseId, course.Path, metadata)
	}

	return c.Status(fiber.StatusCreated).JSON(courseTagResponseHelper([]*models.CourseTag{courseTag})[0])
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) deleteCourseTag(c *fiber.Ctx) error {
	courseId := c.Params("id")
	tagId := c.Params("tagId")

	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Check if scan is in progress
	if api.r.app.CourseScan.IsScanning(courseId) {
		return errorResponse(c, fiber.StatusConflict, "Cannot modify tags while course scan is in progress", nil)
	}

	// Get the course tag to find the tag name
	dbOpts := dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courseId},
			squirrel.Eq{models.COURSE_TAG_TABLE_ID: tagId},
		})
	courseTag, err := api.r.appDao.GetCourseTag(ctx, dbOpts)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up course tag", err)
	}

	// If tag doesn't exist, return 204 (idempotent delete)
	if courseTag == nil {
		return c.Status(fiber.StatusNoContent).Send(nil)
	}

	// Get course to access its path
	course, err := api.getCourseByID(ctx, courseId)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up course", err)
	}

	if course == nil {
		// Course doesn't exist, but tag lookup succeeded - this shouldn't happen
		// But for idempotency, return 204
		return c.Status(fiber.StatusNoContent).Send(nil)
	}

	// Delete tag from DB first (synchronous)
	dbOpts = dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courseId},
			squirrel.Eq{models.COURSE_TAG_TABLE_ID: tagId},
		})

	if err := api.r.appDao.DeleteCourseTags(ctx, dbOpts); err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error deleting course tag", err)
	}

	// Read actual remaining tags from DB AFTER deletion to avoid race conditions
	// This ensures we get the correct state even if other tags were deleted concurrently
	dbOpts = dao.NewOptions().
		WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courseId})
	remainingCourseTags, err := api.r.appDao.ListCourseTags(ctx, dbOpts)
	if err != nil {
		// If reading fails, skip file update - DB delete succeeded
		return c.Status(fiber.StatusNoContent).Send(nil)
	}

	// Build list of remaining tag names
	remainingTags := make([]string, 0, len(remainingCourseTags))
	for _, ct := range remainingCourseTags {
		remainingTags = append(remainingTags, ct.Tag)
	}

	// Queue async file write (fire and forget)
	metadata := &coursemetadata.CourseMetadata{
		Description: "",
		Tags:        remainingTags,
	}
	api.r.app.MetadataWriter.WriteMetadataAsync(courseId, course.Path, metadata)

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) favouriteCourse(c *fiber.Ctx) error {
	courseId := c.Params("id")

	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Verify course exists
	course, err := api.getCourseByID(ctx, courseId)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up course", err)
	}

	if course == nil {
		return errorResponse(c, fiber.StatusNotFound, "Course not found", fmt.Errorf("course not found"))
	}

	courseFavourite := &models.CourseFavourite{
		CourseID: courseId,
		UserID:   principal.UserID,
	}

	if err := api.r.appDao.CreateCourseFavourite(ctx, courseFavourite); err != nil {
		if strings.HasPrefix(err.Error(), "UNIQUE constraint failed") {
			return errorResponse(c, fiber.StatusBadRequest, "Course is already favourited", err)
		}
		return errorResponse(c, fiber.StatusInternalServerError, "Error favouriting course", err)
	}

	return c.Status(fiber.StatusCreated).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (api coursesAPI) unfavouriteCourse(c *fiber.Ctx) error {
	courseId := c.Params("id")

	principal, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	dbOpts := dao.NewOptions().
		WithWhere(squirrel.And{
			squirrel.Eq{models.COURSE_FAVOURITE_TABLE_COURSE_ID: courseId},
			squirrel.Eq{models.COURSE_FAVOURITE_TABLE_USER_ID: principal.UserID},
		})

	if err := api.r.appDao.DeleteCourseFavourites(ctx, dbOpts); err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error unfavouriting course", err)
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getCourseByID retrieves a course by its ID with optional database options
func (api coursesAPI) getCourseByID(ctx context.Context, courseID string, opts ...func(*dao.Options)) (*models.Course, error) {
	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: courseID})
	for _, opt := range opts {
		opt(dbOpts)
	}
	return api.r.appDao.GetCourse(ctx, dbOpts)
}
