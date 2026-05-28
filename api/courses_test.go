package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/cardcache"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/security"
	"github.com/geerew/off-course/utils/types"
	"github.com/gofiber/fiber/v2"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetCourses(t *testing.T) {
	t.Run("200 (empty)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, _ := unmarshalHelper[courseResponse](t, body)
		require.Zero(t, int(paginationResp.TotalItems))
		require.Zero(t, len(paginationResp.Items))
	})

	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		for i := range 5 {
			course := &models.Course{
				Title: fmt.Sprintf("course %d", i+1),
				Path:  fmt.Sprintf("/course %d", i+1),
			}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			time.Sleep(1 * time.Millisecond)
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 5, int(paginationResp.TotalItems))
		require.Len(t, coursesResp, 5)
	})

	t.Run("200 (sort)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 5 {
			course := &models.Course{
				Title: fmt.Sprintf("course %d", i+1),
				Path:  fmt.Sprintf("/course %d", i+1),
			}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
			time.Sleep(1 * time.Millisecond)
		}

		// CREATED_AT ASC
		sortAsc := models.COURSE_TABLE_CREATED_AT + " asc"
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?orderBy="+url.QueryEscape(sortAsc), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 5, int(paginationResp.TotalItems))
		require.Len(t, coursesResp, 5)
		require.Equal(t, courses[0].ID, coursesResp[0].ID)

		// CREATED_AT DESC
		sortDesc := models.COURSE_TABLE_CREATED_AT + " desc"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?orderBy="+url.QueryEscape(sortDesc), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp = unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 5, int(paginationResp.TotalItems))
		require.Len(t, coursesResp, 5)
		require.Equal(t, courses[4].ID, coursesResp[0].ID)

	})

	t.Run("200 (pagination)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 17 {
			course := &models.Course{
				Title: fmt.Sprintf("course %d", i+1),
				Path:  fmt.Sprintf("/course %d", i+1),
			}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
			time.Sleep(1 * time.Millisecond)
		}

		// Page 1 (10 courses)
		params := url.Values{
			"orderBy":                    {models.COURSE_TABLE_CREATED_AT + " asc"},
			pagination.PageQueryParam:    {"1"},
			pagination.PerPageQueryParam: {"10"},
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+params.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 10)
		require.Equal(t, courses[0].ID, coursesResp[0].ID)
		require.Equal(t, courses[9].ID, coursesResp[9].ID)

		// Page 2 (7 courses)
		params = url.Values{
			"orderBy":                    {models.COURSE_TABLE_CREATED_AT + " asc"},
			pagination.PageQueryParam:    {"2"},
			pagination.PerPageQueryParam: {"10"},
		}
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+params.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp = unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 7)
		require.Equal(t, courses[10].ID, coursesResp[0].ID)
		require.Equal(t, courses[16].ID, coursesResp[6].ID)
	})

	t.Run("200 (filter)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		defaultSort := models.COURSE_TABLE_CREATED_AT + " asc"

		courses := []*models.Course{}
		for i := range 6 {
			course := &models.Course{
				Title: fmt.Sprintf("course %d", i+1),
				Path:  fmt.Sprintf("/course %d", i+1),
			}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
			time.Sleep(1 * time.Millisecond)
		}

		// Add asset for each course
		assets := []*models.Asset{}
		for i, c := range courses {
			lesson := &models.Lesson{
				CourseID: c.ID,
				Title:    fmt.Sprintf("lesson %d", i+1),
				Prefix:   sql.NullInt16{Int16: int16(i + 1), Valid: true},
				Module:   "Module 1",
			}
			require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

			asset := &models.Asset{
				CourseID: c.ID,
				LessonID: lesson.ID,
				Title:    "asset 1",
				Prefix:   sql.NullInt16{Int16: 1, Valid: true},
				Module:   "Module 1",
				Type:     types.MustAsset("mp4"),
				Path:     fmt.Sprintf("/course %d/chapter 1/01 asset 1.mp4", i+1),
				FileSize: 1024,
				ModTime:  time.Now().Format(time.RFC3339Nano),
				Hash:     security.RandomString(64),
			}
			require.NoError(t, router.appDao.CreateAsset(ctx, asset))
			assets = append(assets, asset)
		}

		// Set progress (course 1 started, course 5 completed)
		require.NoError(t, router.appDao.UpsertAssetProgress(ctx, &models.AssetProgress{AssetID: assets[0].ID, Position: 10}))
		require.NoError(t, router.appDao.UpsertAssetProgress(ctx, &models.AssetProgress{AssetID: assets[4].ID, Position: 10, Completed: true}))

		// Set availability (courses 1, 3, 5 available)
		for i, c := range courses {
			c.Available = i%2 == 0
			require.NoError(t, router.appDao.UpdateCourse(ctx, c))
		}

		// Set tags
		require.NoError(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: courses[0].ID, Tag: "tag1"}))
		require.NoError(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: courses[0].ID, Tag: "tag2"}))

		require.NoError(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: courses[1].ID, Tag: "tag1"}))
		require.NoError(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: courses[1].ID, Tag: "tag2"}))
		require.NoError(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: courses[1].ID, Tag: "tag3"}))

		require.NoError(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: courses[2].ID, Tag: "tag1"}))

		require.NoError(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: courses[3].ID, Tag: "tag3"}))
		require.NoError(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: courses[3].ID, Tag: "tag4"}))

		// Set favourites (courses 0, 2, 4 are favourited)
		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, &models.CourseFavourite{CourseID: courses[0].ID, UserID: principal.UserID}))
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, &models.CourseFavourite{CourseID: courses[2].ID, UserID: principal.UserID}))
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, &models.CourseFavourite{CourseID: courses[4].ID, UserID: principal.UserID}))

		// No filter
		{
			status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/", nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)

			paginationResp, _ := unmarshalHelper[courseResponse](t, body)
			require.Equal(t, 6, int(paginationResp.TotalItems))
			require.Len(t, paginationResp.Items, 6)
		}

		// Title
		{
			qvals := url.Values{"q": {`title:'course 1' OR title:'course 2' OR title:'course 5'`}, "orderBy": {defaultSort}}
			status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+qvals.Encode(), nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)

			paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
			require.Equal(t, 3, int(paginationResp.TotalItems))
			require.Len(t, paginationResp.Items, 3)
			require.Equal(t, courses[0].ID, coursesResp[0].ID)
			require.Equal(t, courses[1].ID, coursesResp[1].ID)
			require.Equal(t, courses[4].ID, coursesResp[2].ID)
		}

		// Tags
		{
			qvals := url.Values{"q": {"(tag:tag1 AND (tag:tag2 OR tag:tag3)) OR tag:tag4"}, "orderBy": {defaultSort}}
			status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+qvals.Encode(), nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)

			paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
			require.Equal(t, 3, int(paginationResp.TotalItems))
			require.Len(t, paginationResp.Items, 3)
			require.Equal(t, courses[0].ID, coursesResp[0].ID)
			require.Equal(t, courses[1].ID, coursesResp[1].ID)
			require.Equal(t, courses[3].ID, coursesResp[2].ID)
		}

		// Available
		{
			qvals := url.Values{"q": {"available:true"}, "orderBy": {defaultSort}}
			status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+qvals.Encode(), nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)

			paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
			require.Equal(t, 3, int(paginationResp.TotalItems))
			require.Len(t, paginationResp.Items, 3)
			require.Equal(t, courses[0].ID, coursesResp[0].ID)
			require.Equal(t, courses[2].ID, coursesResp[1].ID)
			require.Equal(t, courses[4].ID, coursesResp[2].ID)
		}

		// Progress
		{
			q := `progress:started OR progress:completed OR progress:"not started"`
			qvals := url.Values{"q": {q}, "orderBy": {defaultSort}}
			status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?withUserProgress=true&"+qvals.Encode(), nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)

			paginationResp, _ := unmarshalHelper[courseResponse](t, body)
			require.Equal(t, 6, int(paginationResp.TotalItems))
			require.Len(t, paginationResp.Items, 6)
		}

		// Favourite
		{
			qvals := url.Values{"q": {`favourite:true`}, "orderBy": {defaultSort}}
			status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?withUserProgress=true&"+qvals.Encode(), nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)

			paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
			require.Equal(t, 3, int(paginationResp.TotalItems))
			require.Len(t, coursesResp, 3)
			require.True(t, coursesResp[0].Favourited)
			require.True(t, coursesResp[1].Favourited)
			require.True(t, coursesResp[2].Favourited)
			require.Equal(t, courses[0].ID, coursesResp[0].ID)
			require.Equal(t, courses[2].ID, coursesResp[1].ID)
			require.Equal(t, courses[4].ID, coursesResp[2].ID)
		}

		// Unfavourite
		{
			qvals := url.Values{"q": {`favourite:false`}, "orderBy": {defaultSort}}
			status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?withUserProgress=true&"+qvals.Encode(), nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)

			paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
			require.Equal(t, 3, int(paginationResp.TotalItems))
			require.Len(t, coursesResp, 3)
			require.False(t, coursesResp[0].Favourited)
			require.False(t, coursesResp[1].Favourited)
			require.False(t, coursesResp[2].Favourited)
			require.Equal(t, courses[1].ID, coursesResp[0].ID)
			require.Equal(t, courses[3].ID, coursesResp[1].ID)
			require.Equal(t, courses[5].ID, coursesResp[2].ID)
		}

		// Complex filter
		{
			q := `((title:'course 1' OR title:'course 2') AND available:true AND (tag:tag1 OR tag:tag4)) OR progress:completed`
			qvals := url.Values{"q": {q}, "orderBy": {defaultSort}}
			status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?withUserProgress=true&"+qvals.Encode(), nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, status)

			paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
			require.Equal(t, 2, int(paginationResp.TotalItems))
			require.Len(t, paginationResp.Items, 2)
			require.Equal(t, courses[0].ID, coursesResp[0].ID)
			require.Equal(t, courses[4].ID, coursesResp[1].ID)
		}
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		// Drop the courses table
		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetCourse(t *testing.T) {
	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var courseResp courseResponse
		err = json.Unmarshal(body, &courseResp)
		require.NoError(t, err)
		require.Equal(t, courses[1].ID, courseResp.ID)
	})

	t.Run("404 (not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/invalid", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/invalid", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_CreateCourse(t *testing.T) {
	t.Run("201 (created)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		router.app.FS.MkdirAll("/course 1", os.ModePerm)

		req := httptest.NewRequest(http.MethodPost, "/api/courses/", strings.NewReader(`{"title": "course 1", "path": "/course 1" }`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, status)

		var courseResp courseResponse
		err = json.Unmarshal(body, &courseResp)
		require.NoError(t, err)
		require.NotNil(t, courseResp.ID)
		require.Equal(t, "course 1", courseResp.Title)
		require.Equal(t, "/course 1", courseResp.Path)
		require.True(t, courseResp.Available)
	})

	t.Run("400 (bind error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPost, "/api/courses/", strings.NewReader(`{`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Error parsing data")
	})

	t.Run("400 (invalid data)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		// Missing title
		req := httptest.NewRequest(http.MethodPost, "/api/courses/", strings.NewReader(`{"title": ""}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "A title and path are required")

		// Missing path
		req = httptest.NewRequest(http.MethodPost, "/api/courses/", strings.NewReader(`{"title": "course 1", "path": ""}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err = requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "A title and path are required")

		// Invalid path
		req = httptest.NewRequest(http.MethodPost, "/api/courses/", strings.NewReader(`{"title": "course 1", "path": "/test"}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err = requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Invalid course path")
	})

	t.Run("400 (existing course)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		router.app.FS.MkdirAll("/course 1", os.ModePerm)

		req := httptest.NewRequest(http.MethodPost, "/api/courses/", strings.NewReader(`{"title": "course 1", "path": "/course 1" }`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, status)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "A course with this path already exists")
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TABLE)
		require.NoError(t, err)

		router.app.FS.MkdirAll("/course 1", os.ModePerm)

		req := httptest.NewRequest(http.MethodPost, "/api/courses/", strings.NewReader(`{"title": "course 1", "path": "/course 1" }`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
		require.Contains(t, string(body), "Error creating course")
	})

	// Note: Scan error test removed - scans are now in-memory and don't depend on database tables
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_DeleteCourse(t *testing.T) {
	t.Run("204 (deleted)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/"+courses[1].ID, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: courses[1].ID})
		course, err := router.appDao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Nil(t, course)
	})

	t.Run("204 (deleted with card)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{
			Title:    "course with card",
			Path:     "/course with card",
			CardPath: "/course with card/card.png",
		}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		cardPath := filepath.Join(router.app.Config.DataDir, "cards", course.ID+".webp")
		require.NoError(t, afero.WriteFile(router.app.FS, cardPath, []byte("test card"), os.ModePerm))

		exists, err := afero.Exists(router.app.FS, cardPath)
		require.NoError(t, err)
		require.True(t, exists, "Card should exist before deletion")

		// Delete the course
		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/"+course.ID, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		// Verify course is deleted
		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		deletedCourse, err := router.appDao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Nil(t, deletedCourse)

		exists, err = afero.Exists(router.app.FS, cardPath)
		require.NoError(t, err)
		require.False(t, exists, "Card should be deleted when course is deleted")
	})

	t.Run("204 (not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/invalid", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/invalid", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetCard(t *testing.T) {
	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{
			Title:    "course 1",
			Path:     "/course 1",
			CardPath: "/course 1/card.png",
		}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		require.NoError(t, router.app.FS.MkdirAll(filepath.Dir(course.CardPath), os.ModePerm))
		require.NoError(t, afero.WriteFile(router.app.FS, course.CardPath, []byte("test card"), os.ModePerm))
		_ = router.app.CardCache.OptimizeCard(context.Background(), course.ID, course.CardPath, "")

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/card", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "test card", string(body))
	})

	t.Run("200 (invalid id serves fallback)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		// Invalid course ID - should serve fallback (fallback is created during app initialization)
		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/invalid/card", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status) // Fallback is served, not 404
	})

	t.Run("200 (no card serves fallback)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{
			Title: "course 1",
			Path:  "/course 1",
		}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		// Course has no card - should serve fallback
		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/card", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status) // Fallback is served, not 404
	})

	t.Run("200 (no cache serves original)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{
			Title:    "course 1",
			Path:     "/course 1",
			CardPath: "/course 1/card.png",
		}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))
		require.NoError(t, router.app.FS.MkdirAll(filepath.Dir(course.CardPath), os.ModePerm))
		require.NoError(t, afero.WriteFile(router.app.FS, course.CardPath, []byte("original card"), os.ModePerm))
		_ = router.app.CardCache.OptimizeCard(context.Background(), course.ID, course.CardPath, "")

		serve, err := router.app.CardCache.Get(course.ID)
		require.NoError(t, err)
		require.False(t, serve.Fallback)
		require.Equal(t, course.CardPath, serve.Path)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/card", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "original card", string(body))
	})

	t.Run("200 (missing original serves fallback)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{
			Title:    "course 1",
			Path:     "/course 1",
			CardPath: "/course 1/card.png",
		}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		_ = router.app.CardCache.OptimizeCard(context.Background(), course.ID, course.CardPath, "")

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/card", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
	})

	t.Run("304 (etag)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		cardHash := "abc123deadbeef"
		course := &models.Course{
			Title:    "course etag",
			Path:     "/course etag",
			CardPath: "/course etag/card.png",
			CardHash: cardHash,
		}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))
		require.NoError(t, router.app.FS.MkdirAll(filepath.Dir(course.CardPath), os.ModePerm))
		require.NoError(t, afero.WriteFile(router.app.FS, course.CardPath, []byte("etag card"), os.ModePerm))
		router.app.CardCache.Warm([]cardcache.CourseCardRef{{
			ID: course.ID, CardPath: course.CardPath, CardHash: cardHash,
		}})

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/card", nil)
		req.Header.Set(fiber.HeaderIfNoneMatch, cardcache.FormatETag(cardHash))

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotModified, status)
		require.Empty(t, body)
	})

	t.Run("200 (etag cache headers)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		cardHash := "feedfacecafe"
		course := &models.Course{
			Title:    "course cache",
			Path:     "/course cache",
			CardPath: "/course cache/card.png",
			CardHash: cardHash,
		}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))
		require.NoError(t, router.app.FS.MkdirAll(filepath.Dir(course.CardPath), os.ModePerm))
		require.NoError(t, afero.WriteFile(router.app.FS, course.CardPath, []byte("cache card"), os.ModePerm))
		router.app.CardCache.Warm([]cardcache.CourseCardRef{{
			ID: course.ID, CardPath: course.CardPath, CardHash: cardHash,
		}})

		resp, err := router.Test(httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/card", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, cardcache.FormatETag(cardHash), resp.Header.Get(fiber.HeaderETag))
		require.Contains(t, resp.Header.Get(fiber.HeaderCacheControl), "immutable")
	})

	t.Run("404 (fallback not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		require.NoError(t, router.app.FS.Remove(filepath.Join(router.app.Config.DataDir, "cards", "fallback.webp")))

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/invalid/card", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Fallback card not found")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetLessons(t *testing.T) {
	t.Run("200 (empty)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 2 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/lessons", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, _ := unmarshalHelper[lessonResponse](t, body)
		require.Zero(t, int(paginationResp.TotalItems))
		require.Zero(t, len(paginationResp.Items))
	})

	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		lessons := []*models.Lesson{}
		assets := []*models.Asset{}
		attachments := []*models.Attachment{}

		for i := range 2 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i+1), Path: fmt.Sprintf("/course/%d", i+1)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		// Create 2 lessons, with 1 attachment with 2 assets each for each course
		for _, c := range courses {
			for j := range 2 {
				lesson := &models.Lesson{
					CourseID: c.ID,
					Title:    fmt.Sprintf("lesson %d", j+1),
					Prefix:   sql.NullInt16{Int16: int16(j + 1), Valid: true},
					Module:   "Module 1",
				}
				require.NoError(t, router.appDao.CreateLesson(ctx, lesson))
				lessons = append(lessons, lesson)

				attachment := &models.Attachment{
					CourseID: c.ID,
					LessonID: lesson.ID,
					Title:    fmt.Sprintf("attachment %d", j+1),
					Path:     fmt.Sprintf("/%s/attachment %d", security.RandomString(4), j+1),
				}
				require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
				attachments = append(attachments, attachment)

				for k := range 2 {
					asset := &models.Asset{
						CourseID: c.ID,
						LessonID: lesson.ID,
						Title:    fmt.Sprintf("asset %d", k+1),
						Prefix:   sql.NullInt16{Int16: int16(k + 1), Valid: true},
						Module:   fmt.Sprintf("Chapter %d", k+1),
						Type:     types.MustAsset("mp4"),
						Path:     fmt.Sprintf("/%s/asset %d", security.RandomString(4), k+1),
						FileSize: 1024,
						ModTime:  time.Now().Format(time.RFC3339Nano),
						Hash:     security.RandomString(64),
					}
					require.NoError(t, router.appDao.CreateAsset(ctx, asset))
					assets = append(assets, asset)
					time.Sleep(1 * time.Millisecond)
				}
			}
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/lessons", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, lessonsResp := unmarshalHelper[lessonResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 2)

		require.Equal(t, lessons[2].ID, lessonsResp[0].ID)
		require.Equal(t, lessons[3].ID, lessonsResp[1].ID)

		// Attachments
		require.Len(t, lessonsResp[0].Attachments, 1)
		require.Equal(t, lessonsResp[0].Attachments[0].ID, attachments[2].ID)
		require.Len(t, lessonsResp[1].Attachments, 1)
		require.Equal(t, lessonsResp[1].Attachments[0].ID, attachments[3].ID)

		// Asset 1
		require.Len(t, lessonsResp[0].Assets, 2)
		require.Equal(t, assets[4].ID, lessonsResp[0].Assets[0].ID)
		require.Nil(t, lessonsResp[0].Assets[0].Progress)
		require.Equal(t, assets[5].ID, lessonsResp[0].Assets[1].ID)
		require.Nil(t, lessonsResp[0].Assets[1].Progress)

		// Asset 2
		require.Len(t, lessonsResp[1].Assets, 2)
		require.Equal(t, assets[6].ID, lessonsResp[1].Assets[0].ID)
		require.Nil(t, lessonsResp[1].Assets[0].Progress)
		require.Equal(t, assets[7].ID, lessonsResp[1].Assets[1].ID)
		require.Nil(t, lessonsResp[1].Assets[1].Progress)
	})

	t.Run("200 (withUserProgress)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		lessons := []*models.Lesson{}
		assets := []*models.Asset{}
		attachments := []*models.Attachment{}

		for i := range 2 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i+1), Path: fmt.Sprintf("/course/%d", i+1)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		// Create 2 lessons, with 1 attachment with 2 assets each for each course
		for _, c := range courses {
			for j := range 2 {
				lesson := &models.Lesson{
					CourseID: c.ID,
					Title:    fmt.Sprintf("lesson %d", j+1),
					Prefix:   sql.NullInt16{Int16: int16(j + 1), Valid: true},
					Module:   "Module 1",
				}
				require.NoError(t, router.appDao.CreateLesson(ctx, lesson))
				lessons = append(lessons, lesson)

				attachment := &models.Attachment{
					CourseID: c.ID,
					LessonID: lesson.ID,
					Title:    fmt.Sprintf("attachment %d", j+1),
					Path:     fmt.Sprintf("/%s/attachment %d", security.RandomString(4), j+1),
				}
				require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
				attachments = append(attachments, attachment)

				for k := range 2 {
					asset := &models.Asset{
						CourseID: c.ID,
						LessonID: lesson.ID,
						Title:    fmt.Sprintf("asset %d", k+1),
						Prefix:   sql.NullInt16{Int16: int16(k + 1), Valid: true},
						Module:   fmt.Sprintf("Chapter %d", k+1),
						Type:     types.MustAsset("mp4"),
						Path:     fmt.Sprintf("/%s/asset %d", security.RandomString(4), k+1),
						FileSize: 1024,
						ModTime:  time.Now().Format(time.RFC3339Nano),
						Hash:     security.RandomString(64),
					}
					require.NoError(t, router.appDao.CreateAsset(ctx, asset))
					assets = append(assets, asset)
					time.Sleep(1 * time.Millisecond)
				}
			}
		}

		// ?withUserProgress=true
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/lessons?withUserProgress=true", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, lessonsResp := unmarshalHelper[lessonResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 2)

		require.Equal(t, lessons[2].ID, lessonsResp[0].ID)
		require.Equal(t, lessons[3].ID, lessonsResp[1].ID)

		// Attachments
		require.Len(t, lessonsResp[0].Attachments, 1)
		require.Equal(t, lessonsResp[0].Attachments[0].ID, attachments[2].ID)
		require.Len(t, lessonsResp[1].Attachments, 1)
		require.Equal(t, lessonsResp[1].Attachments[0].ID, attachments[3].ID)

		// Asset 1 - no progress records created, so Progress is nil
		require.Len(t, lessonsResp[0].Assets, 2)
		require.Equal(t, assets[4].ID, lessonsResp[0].Assets[0].ID)
		require.Nil(t, lessonsResp[0].Assets[0].Progress)
		require.Equal(t, assets[5].ID, lessonsResp[0].Assets[1].ID)
		require.Nil(t, lessonsResp[0].Assets[1].Progress)

		// Asset 2
		require.Len(t, lessonsResp[1].Assets, 2)
		require.Equal(t, assets[6].ID, lessonsResp[1].Assets[0].ID)
		require.Nil(t, lessonsResp[1].Assets[0].Progress)
		require.Equal(t, assets[7].ID, lessonsResp[1].Assets[1].ID)
		require.Nil(t, lessonsResp[1].Assets[1].Progress)
	})

	t.Run("200 (orderBy)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 2 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		lessons := []*models.Lesson{}
		for _, c := range courses {
			for j := range 2 {
				lesson := &models.Lesson{
					CourseID: c.ID,
					Title:    fmt.Sprintf("Lesson %d", j+1),
					Prefix:   sql.NullInt16{Int16: int16(j + 1), Valid: true},
					Module:   fmt.Sprintf("Chapter %d", j+1),
				}
				require.NoError(t, router.appDao.CreateLesson(ctx, lesson))
				lessons = append(lessons, lesson)
				time.Sleep(1 * time.Millisecond)
			}
		}

		// CREATED_AT ASC
		sortAsc := models.LESSON_TABLE_CREATED_AT + " asc"
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/lessons/?orderBy="+url.QueryEscape(sortAsc), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, lessonsResp := unmarshalHelper[lessonResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 2)
		require.Equal(t, lessons[2].ID, lessonsResp[0].ID)
		require.Equal(t, lessons[3].ID, lessonsResp[1].ID)

		// CREATED_AT DESC
		sortDesc := models.LESSON_TABLE_CREATED_AT + " desc"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/lessons/?orderBy="+url.QueryEscape(sortDesc), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, lessonsResp = unmarshalHelper[lessonResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 2)
		require.Equal(t, lessons[3].ID, lessonsResp[0].ID)
		require.Equal(t, lessons[2].ID, lessonsResp[1].ID)
	})

	t.Run("200 (pagination)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lessons := []*models.Lesson{}
		for i := range 17 {
			lesson := &models.Lesson{
				CourseID: course.ID,
				Title:    fmt.Sprintf("asset %d", i+1),
				Prefix:   sql.NullInt16{Int16: int16(i + 1), Valid: true},
				Module:   fmt.Sprintf("Chapter %d", i+1),
			}
			require.NoError(t, router.appDao.CreateLesson(ctx, lesson))
			lessons = append(lessons, lesson)
			time.Sleep(1 * time.Millisecond)
		}

		// Get the first page (10 lessons)
		params := url.Values{
			"orderBy":                    {models.LESSON_TABLE_CREATED_AT + " asc"},
			pagination.PageQueryParam:    {"1"},
			pagination.PerPageQueryParam: {"10"},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/?"+params.Encode(), nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, lessonsResp := unmarshalHelper[lessonResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 10)
		require.Equal(t, lessons[0].ID, lessonsResp[0].ID)
		require.Equal(t, lessons[9].ID, lessonsResp[9].ID)

		// Get the second page (7 lessons)
		params = url.Values{
			"orderBy":                    {models.LESSON_TABLE_CREATED_AT + " asc"},
			pagination.PageQueryParam:    {"2"},
			pagination.PerPageQueryParam: {"10"},
		}

		req = httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/?"+params.Encode(), nil)
		status, body, err = requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, lessonsResp = unmarshalHelper[lessonResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 7)
		require.Equal(t, lessons[10].ID, lessonsResp[0].ID)
		require.Equal(t, lessons[16].ID, lessonsResp[6].ID)
	})

	t.Run("500 (asset internal error)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.LESSON_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetLesson(t *testing.T) {
	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lessons := []*models.Lesson{}
		attachments := []*models.Attachment{}

		// Create 2 lessons, with 2 attachments and 2 assets each
		for i := range 2 {
			ag := &models.Lesson{
				CourseID: course.ID,
				Title:    fmt.Sprintf("lesson %d", i+1),
				Prefix:   sql.NullInt16{Int16: int16(i + 1), Valid: true},
				Module:   fmt.Sprintf("Chapter %d", i+1),
			}
			require.NoError(t, router.appDao.CreateLesson(ctx, ag))
			lessons = append(lessons, ag)
			time.Sleep(1 * time.Millisecond)

			// Two assets and attachments per lesson
			for j := range 2 {
				attachment := &models.Attachment{
					CourseID: course.ID,
					LessonID: ag.ID,
					Title:    fmt.Sprintf("attachment %d", j+1),
					Path:     fmt.Sprintf("/%s/attachment %d", security.RandomString(4), j+1),
				}
				require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
				attachments = append(attachments, attachment)

				asset := &models.Asset{
					CourseID:  course.ID,
					LessonID:  ag.ID,
					Title:     fmt.Sprintf("video %d", j+1),
					Prefix:    sql.NullInt16{Int16: ag.Prefix.Int16, Valid: true},
					SubPrefix: sql.NullInt16{Int16: int16(j + 1), Valid: true},
					Module:    ag.Module,
					Type:      types.MustAsset("mp4"),
					Path:      fmt.Sprintf("/course-1/%02d video %d {%02d}.mp4", ag.Prefix.Int16, j+1, j+1),
				}
				require.NoError(t, router.appDao.CreateAsset(ctx, asset))
			}
		}

		target := lessons[1]
		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+target.ID, nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var resp lessonResponse
		require.NoError(t, json.Unmarshal(body, &resp))
		require.Equal(t, target.ID, resp.ID)
		require.Equal(t, target.Title, resp.Title)

		// Attachments
		require.Len(t, resp.Attachments, 2)
		require.Equal(t, attachments[2].ID, resp.Attachments[0].ID)
		require.Equal(t, attachments[3].ID, resp.Attachments[1].ID)

		// assets for lesson 2 (2 total, progress must be nil)
		require.Len(t, resp.Assets, 2)
		for _, a := range resp.Assets {
			require.Nil(t, a.Progress)
		}
	})

	t.Run("200 (found withUserProgress)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lessons := []*models.Lesson{}
		attachments := []*models.Attachment{}

		// Create 2 lessons, with 2 attachments and 2 assets each
		for i := range 2 {
			ag := &models.Lesson{
				CourseID: course.ID,
				Title:    fmt.Sprintf("lesson %d", i+1),
				Prefix:   sql.NullInt16{Int16: int16(i + 1), Valid: true},
				Module:   fmt.Sprintf("Chapter %d", i+1),
			}
			require.NoError(t, router.appDao.CreateLesson(ctx, ag))
			lessons = append(lessons, ag)
			time.Sleep(1 * time.Millisecond)

			for j := range 2 {
				attachment := &models.Attachment{
					CourseID: course.ID,
					LessonID: ag.ID,
					Title:    fmt.Sprintf("attachment %d", j+1),
					Path:     fmt.Sprintf("/%s/attachment %d", security.RandomString(4), j+1),
				}
				require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
				attachments = append(attachments, attachment)

				asset := &models.Asset{
					CourseID:  course.ID,
					LessonID:  ag.ID,
					Title:     fmt.Sprintf("video %d", j+1),
					Prefix:    sql.NullInt16{Int16: ag.Prefix.Int16, Valid: true},
					SubPrefix: sql.NullInt16{Int16: int16(j + 1), Valid: true},
					Module:    ag.Module,
					Type:      types.MustAsset("mp4"),
					Path:      fmt.Sprintf("/course-1/%02d video %d {%02d}.mp4", ag.Prefix.Int16, j+1, j+1),
				}
				require.NoError(t, router.appDao.CreateAsset(ctx, asset))

				// Create asset metadata
				meta := &models.AssetMetadata{
					AssetID: asset.ID,
					VideoMetadata: &models.VideoMetadata{
						DurationSec: 120,
						Container:   "mov,mp4,m4a,3gp,3g2,mj2",
						MIMEType:    "video/mp4",
						SizeBytes:   1024,
						OverallBPS:  200000,
						VideoCodec:  "h264",
						Width:       1280,
						Height:      720,
						FPSNum:      30,
						FPSDen:      1,
					},
					AudioMetadata: nil,
				}
				require.NoError(t, router.appDao.CreateAssetMetadata(ctx, meta))

				// Mark the asset as completed
				assetProgress := &models.AssetProgress{
					AssetID:     asset.ID,
					Position:    120,
					Completed:   true,
					CompletedAt: types.NowDateTime(),
				}
				require.NoError(t, router.appDao.UpsertAssetProgress(ctx, assetProgress))
			}
		}

		target := lessons[1]

		// ?withUserProgress=true
		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+target.ID+"?withUserProgress=true", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var resp lessonResponse
		require.NoError(t, json.Unmarshal(body, &resp))
		require.Equal(t, target.ID, resp.ID)
		require.Equal(t, target.Title, resp.Title)

		// Lesson status
		require.Equal(t, resp.Started, true)
		require.Equal(t, resp.Completed, true)
		require.Equal(t, 2, resp.AssetsCompleted)
		require.Equal(t, 240, resp.TotalVideoDuration)

		// Attachments
		require.Len(t, resp.Attachments, 2)
		require.Equal(t, attachments[2].ID, resp.Attachments[0].ID)
		require.Equal(t, attachments[3].ID, resp.Attachments[1].ID)

		// Assets
		require.Len(t, resp.Assets, 2)
		for _, a := range resp.Assets {
			require.NotNil(t, a.Progress)
		}
	})

	t.Run("404 (invalid lesson for course)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course1 := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course1))

		course2 := &models.Course{Title: "course 2", Path: "/course 2"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course2))

		lesson := &models.Lesson{
			CourseID: course2.ID,
			Title:    "Asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		// Request an lesson that does not belong to the course
		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course1.ID+"/lessons/"+lesson.ID, nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Lesson not found")
	})

	t.Run("404 (lesson not found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/invalid", nil)
		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
	})

	t.Run("500 (lesson internal error)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.LESSON_TABLE)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/invalid", nil)
		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetModules(t *testing.T) {
	t.Run("200 (empty)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 2 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/modules", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var response modulesResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		require.Zero(t, len(response.Modules))
	})

	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		lessons := []*models.Lesson{}
		assets := []*models.Asset{}
		attachments := []*models.Attachment{}

		for i := range 2 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i+1), Path: fmt.Sprintf("/course/%d", i+1)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		// Create 2 lessons, with 1 attachment with 2 assets each for each course
		for _, c := range courses {
			for j := range 2 {
				lesson := &models.Lesson{
					CourseID: c.ID,
					Title:    fmt.Sprintf("lesson %d", j+1),
					Prefix:   sql.NullInt16{Int16: int16(j + 1), Valid: true},
					Module:   fmt.Sprintf("Chapter %d", j+1),
				}
				require.NoError(t, router.appDao.CreateLesson(ctx, lesson))
				lessons = append(lessons, lesson)

				attachment := &models.Attachment{
					CourseID: c.ID,
					LessonID: lesson.ID,
					Title:    fmt.Sprintf("attachment %d", j+1),
					Path:     fmt.Sprintf("%s/attachment %d", c.Path, j+1),
				}
				require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
				attachments = append(attachments, attachment)

				asset := &models.Asset{
					CourseID: c.ID,
					LessonID: lesson.ID,
					Title:    "asset 1",
					Prefix:   sql.NullInt16{Int16: int16(1), Valid: true},
					Module:   fmt.Sprintf("%d Chapter %d", j+1, j+1),
					Type:     types.MustAsset("mp4"),
					Path:     fmt.Sprintf("%s/%d Chapter %d/asset 1", c.Path, j+1, j+1),
					FileSize: 1024,
					ModTime:  time.Now().Format(time.RFC3339Nano),
					Hash:     security.RandomString(64),
				}
				require.NoError(t, router.appDao.CreateAsset(ctx, asset))
				assets = append(assets, asset)
				time.Sleep(1 * time.Millisecond)
			}
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/modules", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var response modulesResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		require.Len(t, response.Modules, 2)

		// Module 1
		require.Equal(t, lessons[2].Module, response.Modules[0].Module)
		require.Len(t, response.Modules[0].Lessons, 1)
		require.Len(t, response.Modules[0].Lessons[0].Assets, 1)
		require.Equal(t, assets[2].Title, response.Modules[0].Lessons[0].Assets[0].Title)
		require.Nil(t, response.Modules[0].Lessons[0].Assets[0].Progress)
		require.Len(t, response.Modules[0].Lessons[0].Attachments, 1)
		require.Equal(t, attachments[2].Title, response.Modules[0].Lessons[0].Attachments[0].Title)

		// Module 2
		require.Equal(t, lessons[3].Module, response.Modules[1].Module)
		require.Len(t, response.Modules[1].Lessons, 1)
		require.Len(t, response.Modules[1].Lessons[0].Assets, 1)
		require.Equal(t, assets[3].Title, response.Modules[1].Lessons[0].Assets[0].Title)
		require.Nil(t, response.Modules[1].Lessons[0].Assets[0].Progress)
		require.Len(t, response.Modules[1].Lessons[0].Attachments, 1)
		require.Equal(t, attachments[3].Title, response.Modules[1].Lessons[0].Attachments[0].Title)
	})

	t.Run("200 (withUserProgress)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		lessons := []*models.Lesson{}
		assets := []*models.Asset{}
		attachments := []*models.Attachment{}

		for i := range 2 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i+1), Path: fmt.Sprintf("/course/%d", i+1)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		// Create 2 lessons, with 1 attachment with 2 assets each for each course
		for _, c := range courses {
			for j := range 2 {
				lesson := &models.Lesson{
					CourseID: c.ID,
					Title:    fmt.Sprintf("lesson %d", j+1),
					Prefix:   sql.NullInt16{Int16: int16(j + 1), Valid: true},
					Module:   fmt.Sprintf("Chapter %d", j+1),
				}
				require.NoError(t, router.appDao.CreateLesson(ctx, lesson))
				lessons = append(lessons, lesson)

				attachment := &models.Attachment{
					CourseID: c.ID,
					LessonID: lesson.ID,
					Title:    fmt.Sprintf("attachment %d", j+1),
					Path:     fmt.Sprintf("%s/attachment %d", c.Path, j+1),
				}
				require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
				attachments = append(attachments, attachment)

				asset := &models.Asset{
					CourseID: c.ID,
					LessonID: lesson.ID,
					Title:    "asset 1",
					Prefix:   sql.NullInt16{Int16: int16(1), Valid: true},
					Module:   fmt.Sprintf("%d Chapter %d", j+1, j+1),
					Type:     types.MustAsset("mp4"),
					Path:     fmt.Sprintf("%s/%d Chapter %d/asset 1", c.Path, j+1, j+1),
					FileSize: 1024,
					ModTime:  time.Now().Format(time.RFC3339Nano),
					Hash:     security.RandomString(64),
				}
				require.NoError(t, router.appDao.CreateAsset(ctx, asset))
				assets = append(assets, asset)
				time.Sleep(1 * time.Millisecond)
			}
		}

		// ?withUserProgress=true
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/modules?withUserProgress=true", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var response modulesResponse
		err = json.Unmarshal(body, &response)
		require.NoError(t, err)
		require.Len(t, response.Modules, 2)

		// Module 1 - no progress records created, so Progress is nil
		require.Equal(t, lessons[2].Module, response.Modules[0].Module)
		require.Len(t, response.Modules[0].Lessons, 1)
		require.Len(t, response.Modules[0].Lessons[0].Assets, 1)
		require.Equal(t, assets[2].Title, response.Modules[0].Lessons[0].Assets[0].Title)
		require.Nil(t, response.Modules[0].Lessons[0].Assets[0].Progress)
		require.Len(t, response.Modules[0].Lessons[0].Attachments, 1)
		require.Equal(t, attachments[2].Title, response.Modules[0].Lessons[0].Attachments[0].Title)

		// Module 2
		require.Equal(t, lessons[3].Module, response.Modules[1].Module)
		require.Len(t, response.Modules[1].Lessons, 1)
		require.Len(t, response.Modules[1].Lessons[0].Assets, 1)
		require.Equal(t, assets[3].Title, response.Modules[1].Lessons[0].Assets[0].Title)
		require.Nil(t, response.Modules[1].Lessons[0].Assets[0].Progress)
		require.Len(t, response.Modules[1].Lessons[0].Attachments, 1)
		require.Equal(t, attachments[3].Title, response.Modules[1].Lessons[0].Attachments[0].Title)
	})

	t.Run("500 (asset internal error)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.LESSON_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/modules/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetAttachments(t *testing.T) {
	t.Run("200 (empty)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, _ := unmarshalHelper[attachmentResponse](t, body)
		require.Zero(t, int(paginationResp.TotalItems))
		require.Zero(t, len(paginationResp.Items))
	})

	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		attachments := []*models.Attachment{}
		for i := range 2 {
			attachment := &models.Attachment{
				CourseID: course.ID,
				LessonID: lesson.ID,
				Title:    fmt.Sprintf("attachment %d", i+1),
				Path:     fmt.Sprintf("/%s/attachment %d", security.RandomString(4), i+1),
			}
			require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
			attachments = append(attachments, attachment)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, attachmentResp := unmarshalHelper[attachmentResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 2)
		require.Equal(t, attachments[0].ID, attachmentResp[0].ID)
		require.Equal(t, attachments[1].ID, attachmentResp[1].ID)
	})

	t.Run("200 (orderBy)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		attachments := []*models.Attachment{}
		for i := range 2 {
			attachment := &models.Attachment{
				CourseID: course.ID,
				LessonID: lesson.ID,
				Title:    fmt.Sprintf("attachment %d", i+1),
				Path:     fmt.Sprintf("/%s/attachment %d", security.RandomString(4), i+1),
			}
			require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
			attachments = append(attachments, attachment)
			time.Sleep(1 * time.Millisecond)
		}

		// CREATED_AT ASC
		sortAsc := models.ATTACHMENT_TABLE_CREATED_AT + " asc"
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments?orderBy="+url.QueryEscape(sortAsc), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, attachmentResp := unmarshalHelper[attachmentResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 2)
		require.Equal(t, attachments[0].ID, attachmentResp[0].ID)
		require.Equal(t, attachments[1].ID, attachmentResp[1].ID)

		// CREATED_AT DESC
		sortDesc := models.ATTACHMENT_TABLE_CREATED_AT + " desc"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments?orderBy="+url.QueryEscape(sortDesc), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, attachmentResp = unmarshalHelper[attachmentResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 2)

		require.Equal(t, attachments[1].ID, attachmentResp[0].ID)
		require.Equal(t, attachments[0].ID, attachmentResp[1].ID)
	})

	t.Run("200 (pagination)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		attachments := []*models.Attachment{}
		for i := range 17 {
			attachment := &models.Attachment{
				CourseID: course.ID,
				LessonID: lesson.ID,
				Title:    fmt.Sprintf("attachment %d", i+1),
				Path:     fmt.Sprintf("/%s/attachment %d", security.RandomString(4), i+1),
			}
			require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))
			attachments = append(attachments, attachment)
			time.Sleep(1 * time.Millisecond)
		}

		// Get the first page (10 attachments)
		params := url.Values{
			"orderBy":                    {models.ATTACHMENT_TABLE_CREATED_AT + " asc"},
			pagination.PageQueryParam:    {"1"},
			pagination.PerPageQueryParam: {"10"},
		}

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments?"+params.Encode(), nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, attachmentResp := unmarshalHelper[attachmentResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 10)
		require.Equal(t, attachments[0].ID, attachmentResp[0].ID)
		require.Equal(t, attachments[9].ID, attachmentResp[9].ID)

		// Get the second page (7 attachments)
		params = url.Values{
			"orderBy":                    {models.ATTACHMENT_TABLE_CREATED_AT + " asc"},
			pagination.PageQueryParam:    {"2"},
			pagination.PerPageQueryParam: {"10"},
		}

		req = httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments?"+params.Encode(), nil)
		status, body, err = requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, attachmentResp = unmarshalHelper[attachmentResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 7)
		require.Equal(t, attachments[10].ID, attachmentResp[0].ID)
		require.Equal(t, attachments[16].ID, attachmentResp[6].ID)
	})

	t.Run("200 (invalid lesson)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/invalid/attachments", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, _ := unmarshalHelper[attachmentResponse](t, body)
		require.Zero(t, int(paginationResp.TotalItems))
		require.Zero(t, len(paginationResp.Items))
	})

	t.Run("200 (invalid lesson for course)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course1 := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course1))

		course2 := &models.Course{Title: "Course 2", Path: "/course/2"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course2))

		lesson := &models.Lesson{
			CourseID: course2.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course1.ID+"/lessons/"+lesson.ID+"/attachments", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, _ := unmarshalHelper[attachmentResponse](t, body)
		require.Zero(t, int(paginationResp.TotalItems))
		require.Zero(t, len(paginationResp.Items))
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetAttachment(t *testing.T) {
	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "attachment 1",
			Path:     fmt.Sprintf("/%s/attachment 1", security.RandomString(4)),
		}
		require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments/"+attachment.ID, nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var respData attachmentResponse
		err = json.Unmarshal(body, &respData)
		require.NoError(t, err)
		require.Equal(t, attachment.ID, respData.ID)
	})

	t.Run("404 (invalid lesson for course)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course1 := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course1))

		course2 := &models.Course{Title: "Course 2", Path: "/course/2"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course2))

		lesson := &models.Lesson{
			CourseID: course2.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course1.ID+"/lessons/"+lesson.ID+"/attachments/invalid", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Attachment not found")
	})

	t.Run("404 (invalid attachment for lessons)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson1 := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson1))

		lesson2 := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 2",
			Prefix:   sql.NullInt16{Int16: 2, Valid: true},
			Module:   "Module 2",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson2))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson1.ID,
			Title:    "attachment 1",
			Path:     fmt.Sprintf("/%s/attachment 1", security.RandomString(4)),
		}
		require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson2.ID+"/attachments/"+attachment.ID, nil)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Attachment not found")
	})

	t.Run("404 (course not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodGet, "/api/courses/invalid/lessons/invalid/attachments/invalid", nil)
		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
	})

	t.Run("404 (lesson not found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/invalid/attachments/invalid", nil)
		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
	})

	t.Run("404 (attachment not found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments/invalid", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Attachment not found")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_ServeAttachment(t *testing.T) {
	t.Run("200 (ok)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "attachment 1",
			Path:     fmt.Sprintf("/%s/attachment 1", security.RandomString(4)),
		}
		require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))

		require.Nil(t, router.app.FS.MkdirAll(filepath.Dir(attachment.Path), os.ModePerm))
		require.Nil(t, afero.WriteFile(router.app.FS, attachment.Path, []byte("hello"), os.ModePerm))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments/"+attachment.ID+"/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "hello", string(body))
	})

	t.Run("400 (invalid path)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "attachment 1",
			Path:     fmt.Sprintf("/%s/attachment 1", security.RandomString(4)),
		}
		require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments/"+attachment.ID+"/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Attachment does not exist")
	})

	t.Run("404 (invalid lesson for course)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course1 := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course1))

		course2 := &models.Course{Title: "Course 2", Path: "/course/2"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course2))

		lesson := &models.Lesson{
			CourseID: course2.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course1.ID+"/lessons/"+lesson.ID+"/attachments/invalid/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Attachment not found")
	})

	t.Run("404 (invalid attachment for lesson)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lessons := []*models.Lesson{}
		for j := range 2 {
			lesson := &models.Lesson{
				CourseID: course.ID,
				Title:    fmt.Sprintf("asset %d", j+1),
				Prefix:   sql.NullInt16{Int16: int16(j + 1), Valid: true},
				Module:   fmt.Sprintf("Chapter %d", j+1),
			}
			require.NoError(t, router.appDao.CreateLesson(ctx, lesson))
			lessons = append(lessons, lesson)
		}

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lessons[0].ID,
			Title:    "attachment 1",
			Path:     fmt.Sprintf("/%s/attachment 1", security.RandomString(4)),
		}
		require.NoError(t, router.appDao.CreateAttachment(ctx, attachment))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lessons[1].ID+"/attachments/"+attachment.ID+"/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Attachment not found")
	})

	t.Run("404 (asset not found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/invalid/attachments/invalid/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Attachment not found")
	})

	t.Run("404 (attachment not found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/attachments/invalid/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Attachment not found")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_ServeAsset(t *testing.T) {
	t.Run("200 (full video)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     fmt.Sprintf("/%s/asset 1", security.RandomString(4)),
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     security.RandomString(64),
		}
		require.NoError(t, router.appDao.CreateAsset(ctx, asset))

		require.Nil(t, router.app.FS.MkdirAll(filepath.Dir(asset.Path), os.ModePerm))
		require.Nil(t, afero.WriteFile(router.app.FS, asset.Path, []byte("video"), os.ModePerm))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, "video", string(body))
	})

	t.Run("200 (stream video)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     fmt.Sprintf("/%s/asset 1", security.RandomString(4)),
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     security.RandomString(64),
		}
		require.NoError(t, router.appDao.CreateAsset(ctx, asset))

		require.Nil(t, router.app.FS.MkdirAll(filepath.Dir(asset.Path), os.ModePerm))
		require.Nil(t, afero.WriteFile(router.app.FS, asset.Path, []byte("video"), os.ModePerm))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/serve", nil)
		req.Header.Add("Range", "bytes=0-")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusPartialContent, status)
		require.Equal(t, "video", string(body))
	})

	t.Run("400 (invalid path)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     fmt.Sprintf("/%s/asset 1", security.RandomString(4)),
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     security.RandomString(64),
		}
		require.NoError(t, router.appDao.CreateAsset(ctx, asset))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Asset does not exist")
	})

	t.Run("400 (invalid video range)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     fmt.Sprintf("/%s/asset 1", security.RandomString(4)),
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     security.RandomString(64),
		}
		require.NoError(t, router.appDao.CreateAsset(ctx, asset))

		require.Nil(t, router.app.FS.MkdirAll(filepath.Dir(asset.Path), os.ModePerm))
		require.Nil(t, afero.WriteFile(router.app.FS, asset.Path, []byte("video"), os.ModePerm))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/serve", nil)
		req.Header.Add("Range", "bytes=10-1")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Range start cannot be greater than end")
	})

	t.Run("404 (invalid asset for course)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course1 := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course1))

		course2 := &models.Course{Title: "Course 2", Path: "/course/2"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course2))

		lesson := &models.Lesson{
			CourseID: course2.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course2.ID,
			LessonID: lesson.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     fmt.Sprintf("/%s/asset 1", security.RandomString(4)),
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     security.RandomString(64),
		}
		require.NoError(t, router.appDao.CreateAsset(ctx, asset))

		req := httptest.NewRequest(http.MethodGet, "/api/courses/"+course1.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Asset not found")
	})

	t.Run("404 (not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodGet, "/api/courses/invalid/lessons/invalid/assets/invalid/serve", nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Asset not found")
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.ASSET_TABLE)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/courses/invalid/lessons/invalid/assets/invalid/serve", nil)
		req.Header.Set("Content-Type", "application/json")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
		require.Contains(t, string(body), "Error looking up asset")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_UpdateAssetProgress(t *testing.T) {
	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     fmt.Sprintf("/%s/asset 1", security.RandomString(4)),
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     security.RandomString(64),
		}
		require.NoError(t, router.appDao.CreateAsset(ctx, asset))

		// Update video position
		assetProgress := &assetProgressRequest{
			Position: 45,
		}

		data, err := json.Marshal(assetProgress)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/progress", strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		dbOpts := dao.NewOptions().WithUserProgress().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: asset.ID})
		assetResult, err := router.appDao.GetAsset(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, assetResult)
		require.NotNil(t, assetResult.Progress)
		require.Equal(t, 45, assetResult.Progress.Position)
		require.False(t, assetResult.Progress.Completed)
		require.True(t, assetResult.Progress.CompletedAt.IsZero())

		// Set completed to true
		assetProgress.Completed = true

		data, err = json.Marshal(assetProgress)
		require.NoError(t, err)

		req = httptest.NewRequest(http.MethodPut, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/progress", strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")

		status, _, err = requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		assetResult, err = router.appDao.GetAsset(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, assetResult)
		require.NotNil(t, assetResult.Progress)
		require.Equal(t, 45, assetResult.Progress.Position)
		require.True(t, assetResult.Progress.Completed)
		require.False(t, assetResult.Progress.CompletedAt.IsZero())

		// Set video position to 10 and completed to false
		assetProgress.Position = 10
		assetProgress.Completed = false

		data, err = json.Marshal(assetProgress)
		require.NoError(t, err)

		req = httptest.NewRequest(http.MethodPut, "/api/courses/"+course.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/progress", strings.NewReader(string(data)))
		req.Header.Set("Content-Type", "application/json")

		status, _, err = requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		assetResult, err = router.appDao.GetAsset(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, assetResult)
		require.NotNil(t, assetResult.Progress)
		require.Equal(t, 10, assetResult.Progress.Position)
		require.False(t, assetResult.Progress.Completed)
		require.True(t, assetResult.Progress.CompletedAt.IsZero())
	})

	t.Run("400 (invalid data)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPut, "/api/courses/invalid/lessons/invalid/assets/invalid/progress", strings.NewReader(`bob`))
		req.Header.Set("Content-Type", "application/json")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Error parsing data")
	})

	t.Run("404 (asset not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPut, "/api/courses/invalid/lessons/invalid/assets/invalid/progress", strings.NewReader(`{"Position": 10}`))
		req.Header.Set("Content-Type", "application/json")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Asset not found")
	})

	// Do we care about this case?
	// t.Run("404 (invalid asset for course)", func(t *testing.T) {
	// 	router, ctx := setup(t, "admin", types.UserRoleAdmin)

	// 	course1 := &models.Course{Title: "Course 1", Path: "/Course 1"}
	// 	require.NoError(t, router.appDao.CreateCourse(ctx, course1))

	// 	course2 := &models.Course{Title: "Course 2", Path: "/course/2"}
	// 	require.NoError(t, router.appDao.CreateCourse(ctx, course2))

	// 	lesson := &models.Lesson{
	// 		CourseID: course2.ID,
	// 		Title:    "lesson 1",
	// 		Prefix:   sql.NullInt16{Int16: 1, Valid: true},
	// 		Module:   "Module 1",
	// 	}
	// 	require.NoError(t, router.appDao.CreateLesson(ctx, lesson))

	// 	asset := &models.Asset{
	// 		CourseID: course2.ID,
	// 		LessonID: lesson.ID,
	// 		Title:    "asset 1",
	// 		Prefix:   sql.NullInt16{Int16: 1, Valid: true},
	// 		Module:   "Module 1",
	// 		Type:     types.MustAsset("mp4"),
	// 		Path:     fmt.Sprintf("/%s/asset 1", security.RandomString(4)),
	// 		FileSize: 1024,
	// 		ModTime:  time.Now().Format(time.RFC3339Nano),
	// 		Hash:     security.RandomString(64),
	// 	}
	// 	require.NoError(t, router.appDao.CreateAsset(ctx, asset))

	// 	req := httptest.NewRequest(http.MethodPut, "/api/courses/"+course1.ID+"/lessons/"+lesson.ID+"/assets/"+asset.ID+"/progress", strings.NewReader(`{"Position": 10}`))
	// 	req.Header.Set("Content-Type", "application/json")

	// 	status, body, err := requestHelper(t, router, req)
	// 	require.NoError(t, err)
	// 	require.Equal(t, http.StatusNotFound, status)
	// 	require.Contains(t, string(body), "Asset not found")
	// })
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetTags(t *testing.T) {
	t.Run("200 (empty)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/tags", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var tags []courseTagResponse
		err = json.Unmarshal(body, &tags)
		require.NoError(t, err)
		require.Zero(t, len(tags))
	})

	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tagOptions := []string{"Go", "C", "JavaScript", "TypeScript", "Java", "Python"}

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)

			for _, tag := range tagOptions {
				tag := &models.CourseTag{CourseID: course.ID, Tag: tag}
				require.NoError(t, router.appDao.CreateCourseTag(ctx, tag))
			}
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+courses[1].ID+"/tags", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var tags []courseTagResponse
		err = json.Unmarshal(body, &tags)
		require.NoError(t, err)
		require.Len(t, tags, 6)
		require.Equal(t, "C", tags[0].Tag)
		require.Equal(t, "TypeScript", tags[5].Tag)
	})

	t.Run("200 (course not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/invalid/tags", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var tags []courseTagResponse
		err = json.Unmarshal(body, &tags)
		require.NoError(t, err)
		require.Zero(t, len(tags))
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TAG_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/"+course.ID+"/tags/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_CreateTag(t *testing.T) {
	t.Run("201 (created)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		req := httptest.NewRequest(http.MethodPost, "/api/courses/"+course.ID+"/tags", strings.NewReader(`{"tag": "Go" }`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, status)

		var courseTagResp courseTagResponse
		err = json.Unmarshal(body, &courseTagResp)
		require.NoError(t, err)
		require.NotNil(t, courseTagResp.ID)
		require.Equal(t, "Go", courseTagResp.Tag)
	})

	t.Run("400 (bind error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPost, "/api/courses/invalid/tags", strings.NewReader(`{`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Error parsing data")
	})

	t.Run("400 (invalid data)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		req := httptest.NewRequest(http.MethodPost, "/api/courses/"+course.ID+"/tags", strings.NewReader(`{"tag": ""}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "A tag is required")
	})

	t.Run("400 (existing tag)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		req := httptest.NewRequest(http.MethodPost, "/api/courses/"+course.ID+"/tags", strings.NewReader(`{"tag": "Go"}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, status)

		// Create the tag again
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "A tag for this course already exists")
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TAG_TABLE)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/courses/"+course.ID+"/tags", strings.NewReader(`{"tag": "Go"}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
		require.Contains(t, string(body), "Error creating course tag")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_DeleteTag(t *testing.T) {
	t.Run("204 (deleted)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)

			for j := range 3 {
				tag := &models.CourseTag{CourseID: course.ID, Tag: fmt.Sprintf("Tag %d", j)}
				require.NoError(t, router.appDao.CreateCourseTag(ctx, tag))
			}
		}

		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courses[1].ID})
		records, err := router.appDao.ListCourseTags(ctx, dbOpts)
		require.NoError(t, err)
		require.Len(t, records, 3)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/"+courses[1].ID+"/tags/"+records[1].ID, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		records, err = router.appDao.ListCourseTags(ctx, dbOpts)
		require.NoError(t, err)
		require.Len(t, records, 2)
	})

	t.Run("204 (not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/invalid/tags/invalid", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)
	})

	t.Run("204 (invalid tag for course)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course1 := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course1))

		tag1 := &models.CourseTag{CourseID: course1.ID, Tag: "Go"}
		require.NoError(t, router.appDao.CreateCourseTag(ctx, tag1))

		course2 := &models.Course{Title: "course 2", Path: "/course 2"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course2))

		tag2 := &models.CourseTag{CourseID: course2.ID, Tag: "C"}
		require.NoError(t, router.appDao.CreateCourseTag(ctx, tag2))

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/"+course1.ID+"/tags/"+tag2.ID, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_ID: tag1.ID})
		record, err := router.appDao.GetCourseTag(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)

		dbOpts = dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_ID: tag2.ID})
		record, err = router.appDao.GetCourseTag(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TAG_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/invalid/tags/invalid", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_FavouriteCourse(t *testing.T) {
	t.Run("201 (created)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodPost, "/api/courses/"+course.ID+"/favourite", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, status)

		// Verify it was created
		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		dbOpts := dao.NewOptions().WithWhere(squirrel.And{
			squirrel.Eq{models.COURSE_FAVOURITE_TABLE_COURSE_ID: course.ID},
			squirrel.Eq{models.COURSE_FAVOURITE_TABLE_USER_ID: principal.UserID},
		})
		record, err := router.appDao.GetCourseFavourite(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, course.ID, record.CourseID)
		require.Equal(t, principal.UserID, record.UserID)
	})

	t.Run("400 (already favourited)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: principal.UserID}
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, courseFavourite))

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodPost, "/api/courses/"+course.ID+"/favourite", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Course is already favourited")
	})

	t.Run("401 (unauthorized)", func(t *testing.T) {
		router, _ := setupNoAuth(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(context.Background(), course))

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodPost, "/api/courses/"+course.ID+"/favourite", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Contains(t, string(body), "Missing principal")
	})

	t.Run("404 (course not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodPost, "/api/courses/invalid/favourite", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Course not found")
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_FAVOURITE_TABLE)
		require.NoError(t, err)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodPost, "/api/courses/"+course.ID+"/favourite", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
		require.Contains(t, string(body), "Error favouriting course")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_UnfavouriteCourse(t *testing.T) {
	t.Run("204 (deleted)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: principal.UserID}
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, courseFavourite))

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/"+course.ID+"/favourite", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		// Verify it was deleted
		dbOpts := dao.NewOptions().WithWhere(squirrel.And{
			squirrel.Eq{models.COURSE_FAVOURITE_TABLE_COURSE_ID: course.ID},
			squirrel.Eq{models.COURSE_FAVOURITE_TABLE_USER_ID: principal.UserID},
		})
		record, err := router.appDao.GetCourseFavourite(ctx, dbOpts)
		require.NoError(t, err)
		require.Nil(t, record)
	})

	t.Run("204 (not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(context.Background(), course))

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/"+course.ID+"/favourite", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)
	})

	t.Run("401 (unauthorized)", func(t *testing.T) {
		router, _ := setupNoAuth(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(context.Background(), course))

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/courses/"+course.ID+"/favourite", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, status)
		require.Contains(t, string(body), "Missing principal")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourses_GetCourses_FavouriteFilter(t *testing.T) {
	t.Run("200 (favourited)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		// Favourite first two courses
		courseFavourite1 := &models.CourseFavourite{CourseID: courses[0].ID, UserID: principal.UserID}
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, courseFavourite1))
		courseFavourite2 := &models.CourseFavourite{CourseID: courses[1].ID, UserID: principal.UserID}
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, courseFavourite2))

		query := url.Values{}
		query.Set("q", `favourite:"true"`)
		query.Set("withUserProgress", "true")

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+query.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, coursesResp, 2)
		require.True(t, coursesResp[0].Favourited)
		require.True(t, coursesResp[1].Favourited)
	})

	t.Run("200 (unfavourited)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course %d", i)}
			require.NoError(t, router.appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		// Favourite first course only
		courseFavourite := &models.CourseFavourite{CourseID: courses[0].ID, UserID: principal.UserID}
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, courseFavourite))

		query := url.Values{}
		query.Set("q", `favourite:"false"`)
		query.Set("withUserProgress", "true")

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+query.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, coursesResp, 2)
		require.False(t, coursesResp[0].Favourited)
		require.False(t, coursesResp[1].Favourited)
	})

	t.Run("200 (favourited field included)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: principal.UserID}
		require.NoError(t, router.appDao.CreateCourseFavourite(ctx, courseFavourite))

		query := url.Values{}
		query.Set("withUserProgress", "true")

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+query.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 1, int(paginationResp.TotalItems))
		require.Len(t, coursesResp, 1)
		require.True(t, coursesResp[0].Favourited)
	})

	t.Run("200 (favourited field false when not favourited)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course))

		query := url.Values{}
		query.Set("withUserProgress", "true")

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/courses/?"+query.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, coursesResp := unmarshalHelper[courseResponse](t, body)
		require.Equal(t, 1, int(paginationResp.TotalItems))
		require.Len(t, coursesResp, 1)
		require.False(t, coursesResp[0].Favourited)
	})
}
