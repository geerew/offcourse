package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/coursemetadata"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestTags_GetTags(t *testing.T) {
	t.Run("200 (empty)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, _ := unmarshalHelper[tagResponse](t, body)
		require.Zero(t, int(paginationResp.TotalItems))
		require.Zero(t, len(paginationResp.Items))
	})

	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.Nil(t, router.appDao.CreateCourse(ctx, course))

		for i := range 5 {
			tag := &models.Tag{Tag: fmt.Sprintf("Tag %02d", i)}
			require.Nil(t, router.appDao.CreateTag(ctx, tag))

			courseTag := models.CourseTag{CourseID: course.ID, TagID: tag.ID}
			require.Nil(t, router.appDao.CreateCourseTag(ctx, &courseTag))
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp := unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 5, int(paginationResp.TotalItems))
		require.Len(t, tagsResp, 5)
		require.Equal(t, 1, tagsResp[0].CourseCount)
	})

	t.Run("200 (orderBy)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		course1 := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.Nil(t, router.appDao.CreateCourse(ctx, course1))

		course2 := &models.Course{Title: "Course 2", Path: "/Course 2"}
		require.Nil(t, router.appDao.CreateCourse(ctx, course2))

		tag_keys := []string{"JavaScript", "Python", "Java", "Ruby", "PHP"}
		for _, tag := range tag_keys {
			require.Nil(t, router.appDao.CreateTag(ctx, &models.Tag{Tag: tag}))
			require.Nil(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: course1.ID, Tag: tag}))
			require.Nil(t, router.appDao.CreateCourseTag(ctx, &models.CourseTag{CourseID: course2.ID, Tag: tag}))
			time.Sleep(1 * time.Millisecond)
		}

		// CREATED_AT ASC
		q := "sort:\"" + models.TAG_TABLE_CREATED_AT + " asc\""
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp := unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 5, int(paginationResp.TotalItems))
		require.Len(t, tagsResp, 5)
		require.Equal(t, "JavaScript", tagsResp[0].Tag)
		require.Equal(t, "PHP", tagsResp[4].Tag)

		// CREATED_AT DESC
		q = "sort:\"" + models.TAG_TABLE_CREATED_AT + " desc\""
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp = unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 5, int(paginationResp.TotalItems))
		require.Len(t, tagsResp, 5)
		require.Equal(t, "PHP", tagsResp[0].Tag)
		require.Equal(t, "JavaScript", tagsResp[4].Tag)
	})

	t.Run("200 (filter)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tag_keys := []string{"slightly", "light", "lighter", "highlight", "ghoul", "lightning", "delight"}

		for _, tag := range tag_keys {
			require.Nil(t, router.appDao.CreateTag(ctx, &models.Tag{Tag: tag}))
			time.Sleep(1 * time.Millisecond)
		}

		// Filter `invalid`
		q := "invalid sort:special"
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp := unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 0, int(paginationResp.TotalItems))
		require.Zero(t, tagsResp)

		// Filter by `li`
		q = "li sort:special"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp = unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 6, int(paginationResp.TotalItems))
		require.Len(t, tagsResp, 6)
		require.Equal(t, "light", tagsResp[0].Tag)
		require.Equal(t, "lighter", tagsResp[1].Tag)
		require.Equal(t, "lightning", tagsResp[2].Tag)
		require.Equal(t, "delight", tagsResp[3].Tag)
		require.Equal(t, "highlight", tagsResp[4].Tag)
		require.Equal(t, "slightly", tagsResp[5].Tag)

		// Filter by `gh`
		q = "gh sort:special"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp = unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 7, int(paginationResp.TotalItems))
		require.Len(t, tagsResp, 7)
		require.Equal(t, "ghoul", tagsResp[0].Tag)
		require.Equal(t, "delight", tagsResp[1].Tag)
		require.Equal(t, "highlight", tagsResp[2].Tag)
		require.Equal(t, "light", tagsResp[3].Tag)
		require.Equal(t, "lighter", tagsResp[4].Tag)
		require.Equal(t, "lightning", tagsResp[5].Tag)
		require.Equal(t, "slightly", tagsResp[6].Tag)

		// Filter by `slight`
		q = "slight sort:special"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp = unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 1, int(paginationResp.TotalItems))
		require.Len(t, tagsResp, 1)
		require.Equal(t, "slightly", tagsResp[0].Tag)

		// Case insensitive
		tag := &models.Tag{Tag: "Slight"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag))

		q = "SLigHt sort:special"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp = unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 2, int(paginationResp.TotalItems))
		require.Len(t, tagsResp, 2)
		require.Equal(t, "Slight", tagsResp[0].Tag)
		require.Equal(t, "slightly", tagsResp[1].Tag)
	})

	t.Run("200 (pagination)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tags := []*models.Tag{}
		for i := range 17 {
			tag := &models.Tag{Tag: fmt.Sprintf("Tag %02d", i+1)}
			require.Nil(t, router.appDao.CreateTag(ctx, tag))
			tags = append(tags, tag)
		}

		// Get the first page (10 tags)
		params := url.Values{
			"q":                          {"sort:\"" + models.TAG_TABLE_TAG + " asc\""},
			pagination.PageQueryParam:    {"1"},
			pagination.PerPageQueryParam: {"10"},
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?"+params.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp := unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 10)
		require.Equal(t, tags[0].Tag, tagsResp[0].Tag)
		require.Equal(t, tags[9].Tag, tagsResp[9].Tag)

		// Get the second page (7 tags)
		params = url.Values{
			"q":                          {"sort:\"" + models.TAG_TABLE_TAG + " asc\""},
			pagination.PageQueryParam:    {"2"},
			pagination.PerPageQueryParam: {"10"},
		}
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/?"+params.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, tagsResp = unmarshalHelper[tagResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 7)
		require.Equal(t, tags[10].Tag, tagsResp[0].Tag)
		require.Equal(t, tags[16].Tag, tagsResp[6].Tag)
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		// Drop the courses table
		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.TAG_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestTags_GetTagNames(t *testing.T) {
	t.Run("200 (empty)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var resp []string
		require.Nil(t, json.Unmarshal(body, &resp))
		require.Zero(t, len(resp))
	})

	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		for i := range 5 {
			tag := &models.Tag{Tag: fmt.Sprintf("Tag %02d", i)}
			require.Nil(t, router.appDao.CreateTag(ctx, tag))
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var resp []string
		require.Nil(t, json.Unmarshal(body, &resp))
		require.Len(t, resp, 5)
	})

	t.Run("200 (orderBy)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tag_keys := []string{"JavaScript", "Python", "Java", "Ruby", "PHP"}
		for _, tag := range tag_keys {
			require.Nil(t, router.appDao.CreateTag(ctx, &models.Tag{Tag: tag}))
			time.Sleep(1 * time.Millisecond)
		}

		// CREATED_AT ASC
		q := "sort:\"" + models.TAG_TABLE_CREATED_AT + " asc\""
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var resp []string
		require.Nil(t, json.Unmarshal(body, &resp))
		require.Len(t, resp, 5)
		require.Equal(t, "JavaScript", resp[0])
		require.Equal(t, "PHP", resp[4])

		// CREATED_AT DESC
		q = "sort:\"" + models.TAG_TABLE_CREATED_AT + " desc\""
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		require.Nil(t, json.Unmarshal(body, &resp))
		require.Len(t, resp, 5)
		require.Equal(t, "PHP", resp[0])
		require.Equal(t, "JavaScript", resp[4])
	})

	t.Run("200 (filter)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tag_keys := []string{"slightly", "light", "lighter", "highlight", "ghoul", "lightning", "delight"}

		for _, tag := range tag_keys {
			require.Nil(t, router.appDao.CreateTag(ctx, &models.Tag{Tag: tag}))
			time.Sleep(1 * time.Millisecond)
		}

		// Filter `invalid`
		q := "invalid sort:special"
		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var resp []string
		require.Nil(t, json.Unmarshal(body, &resp))
		require.Empty(t, resp)

		// Filter by `li`
		q = "li sort:special"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		require.Nil(t, json.Unmarshal(body, &resp))
		require.Len(t, resp, 6)
		require.Equal(t, "light", resp[0])
		require.Equal(t, "lighter", resp[1])
		require.Equal(t, "lightning", resp[2])
		require.Equal(t, "delight", resp[3])
		require.Equal(t, "highlight", resp[4])
		require.Equal(t, "slightly", resp[5])

		// Filter by `gh`
		q = "gh sort:special"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		require.Nil(t, json.Unmarshal(body, &resp))
		require.Len(t, resp, 7)
		require.Equal(t, "ghoul", resp[0])
		require.Equal(t, "delight", resp[1])
		require.Equal(t, "highlight", resp[2])
		require.Equal(t, "light", resp[3])
		require.Equal(t, "lighter", resp[4])
		require.Equal(t, "lightning", resp[5])
		require.Equal(t, "slightly", resp[6])

		// Filter by `slight`
		q = "slight sort:special"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		require.Nil(t, json.Unmarshal(body, &resp))
		require.Len(t, resp, 1)
		require.Equal(t, "slightly", resp[0])

		// Case insensitive
		tag := &models.Tag{Tag: "Slight"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag))

		q = "SLigHt sort:special"
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names?q="+url.QueryEscape(q), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		require.Nil(t, json.Unmarshal(body, &resp))
		require.Len(t, resp, 2)
		require.Equal(t, "Slight", resp[0])
		require.Equal(t, "slightly", resp[1])
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		// Drop the courses table
		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.TAG_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/names", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestTags_GetTag(t *testing.T) {
	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tag1 := &models.Tag{Tag: "Go"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag1))

		tag2 := &models.Tag{Tag: "PHP"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag2))

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/Go", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var tagResp tagResponse
		err = json.Unmarshal(body, &tagResp)
		require.NoError(t, err)
		require.Equal(t, tag1.ID, tagResp.ID)
		require.Zero(t, tagResp.CourseCount)

		// Case insensitive
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/pHp", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		err = json.Unmarshal(body, &tagResp)
		require.NoError(t, err)
		require.Equal(t, tag2.ID, tagResp.ID)
		require.Zero(t, tagResp.CourseCount)

		// Add a course to the tag
		course := &models.Course{Title: "Course 1", Path: "/Course 1"}
		require.Nil(t, router.appDao.CreateCourse(ctx, course))

		courseTag := models.CourseTag{CourseID: course.ID, TagID: tag1.ID}
		require.Nil(t, router.appDao.CreateCourseTag(ctx, &courseTag))

		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/Go", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		err = json.Unmarshal(body, &tagResp)
		require.NoError(t, err)
		require.Equal(t, tag1.ID, tagResp.ID)
		require.Equal(t, 1, tagResp.CourseCount)
	})

	t.Run("404 (not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/test", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Tag not found")
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.TAG_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/tags/test", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestTags_CreateTag(t *testing.T) {
	t.Run("201 (created)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPost, "/api/tags/", strings.NewReader(`{"tag": "test"}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, status)

		var tagResp tagResponse
		err = json.Unmarshal(body, &tagResp)
		require.NoError(t, err)
		require.NotNil(t, tagResp.ID)
		require.Equal(t, "test", tagResp.Tag)
	})

	t.Run("400 (bind error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPost, "/api/tags/", strings.NewReader(`{`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Error parsing data")
	})

	t.Run("400 (invalid data)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPost, "/api/tags/", strings.NewReader(`{"tag": ""}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "A tag is required")
	})

	t.Run("400 (existing tag)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPost, "/api/tags/", strings.NewReader(`{"tag": "test"}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, status)

		// Create again
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Tag already exists")
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.TAG_TABLE)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPost, "/api/tags/", strings.NewReader(`{"tag": "test"}`))
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
		require.Contains(t, string(body), "Error creating tag")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestTags_UpdateTag(t *testing.T) {
	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tag1 := &models.Tag{Tag: "Go"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag1))

		// Update from `Go` to `go`
		req := httptest.NewRequest(http.MethodPut, "/api/tags/"+tag1.ID, strings.NewReader(`{"tag":"go"}`))
		req.Header.Set("Content-Type", "application/json")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var tagResp tagResponse
		err = json.Unmarshal(body, &tagResp)
		require.NoError(t, err)
		require.Equal(t, tag1.ID, tagResp.ID)
		require.Equal(t, "go", tagResp.Tag)

		// Update from `go` to `Golang`
		req = httptest.NewRequest(http.MethodPut, "/api/tags/"+tag1.ID, strings.NewReader(`{"tag":"Golang"}`))
		req.Header.Set("Content-Type", "application/json")

		status, body, err = requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		err = json.Unmarshal(body, &tagResp)
		require.NoError(t, err)
		require.Equal(t, tag1.ID, tagResp.ID)
		require.Equal(t, "Golang", tagResp.Tag)
	})

	t.Run("400 (invalid data)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPut, "/api/tags/test", strings.NewReader(`'`))
		req.Header.Set("Content-Type", "application/json")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Error parsing data")
	})

	t.Run("404 (not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodPut, "/api/tags/invalid", strings.NewReader(`{"tag":"go"}`))
		req.Header.Set("Content-Type", "application/json")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
		require.Contains(t, string(body), "Tag not found")
	})

	t.Run("400 (duplicate)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tag1 := &models.Tag{Tag: "Go"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag1))

		tag2 := &models.Tag{Tag: "PHP"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag2))

		// Update from `Go` to `PHP`
		req := httptest.NewRequest(http.MethodPut, "/api/tags/"+tag1.ID, strings.NewReader(`{"tag":"PHP"}`))
		req.Header.Set("Content-Type", "application/json")

		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Tag already exists")

		// Update from `Go` to `php` (case insensitive)
		req = httptest.NewRequest(http.MethodPut, "/api/tags/"+tag1.ID, strings.NewReader(`{"tag":"php"}`))
		req.Header.Set("Content-Type", "application/json")

		status, body, err = requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)
		require.Contains(t, string(body), "Tag already exists")

	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.TAG_TABLE)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodPut, "/api/tags/test", strings.NewReader(`{"id": "1234567"}`))
		req.Header.Set("Content-Type", "application/json")

		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestTags_DeleteTag(t *testing.T) {
	t.Run("204 (deleted)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		tag1 := &models.Tag{Tag: "Go"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag1))

		tag2 := &models.Tag{Tag: "PHP"}
		require.Nil(t, router.appDao.CreateTag(ctx, tag2))

		// Delete tag `Go`
		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/tags/"+tag1.ID, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		count, err := router.appDao.CountTags(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("204 (not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/tags/test", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		_, err := router.app.DbManager.DataDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.TAG_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/tags/test", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})

	t.Run("204 (deleted with oc.json update)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		// Create courses
		course1 := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course1))
		require.NoError(t, router.app.AppFs.Fs.MkdirAll(course1.Path, 0755))

		course2 := &models.Course{Title: "Course 2", Path: "/course-2"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course2))
		require.NoError(t, router.app.AppFs.Fs.MkdirAll(course2.Path, 0755))

		// Create tags
		tag1 := &models.Tag{Tag: "Go"}
		require.NoError(t, router.appDao.CreateTag(ctx, tag1))

		tag2 := &models.Tag{Tag: "PHP"}
		require.NoError(t, router.appDao.CreateTag(ctx, tag2))

		tag3 := &models.Tag{Tag: "Python"}
		require.NoError(t, router.appDao.CreateTag(ctx, tag3))

		// Add tags to courses
		// Course 1: Go, PHP
		course1Tag1 := &models.CourseTag{CourseID: course1.ID, TagID: tag1.ID}
		require.NoError(t, router.appDao.CreateCourseTag(ctx, course1Tag1))
		course1Tag2 := &models.CourseTag{CourseID: course1.ID, TagID: tag2.ID}
		require.NoError(t, router.appDao.CreateCourseTag(ctx, course1Tag2))

		// Course 2: Go, Python
		course2Tag1 := &models.CourseTag{CourseID: course2.ID, TagID: tag1.ID}
		require.NoError(t, router.appDao.CreateCourseTag(ctx, course2Tag1))
		course2Tag3 := &models.CourseTag{CourseID: course2.ID, TagID: tag3.ID}
		require.NoError(t, router.appDao.CreateCourseTag(ctx, course2Tag3))

		// Delete tag "Go" (tag1)
		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodDelete, "/api/tags/"+tag1.ID, nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, status)

		// Wait for async file writes to complete
		time.Sleep(200 * time.Millisecond)

		// Verify oc.json files were updated
		// Course 1 should only have PHP now
		metadata1, err := coursemetadata.ReadMetadata(router.app.AppFs.Fs, course1.Path)
		require.NoError(t, err)
		require.NotNil(t, metadata1)
		require.Equal(t, []string{"PHP"}, metadata1.Tags)

		// Course 2 should only have Python now
		metadata2, err := coursemetadata.ReadMetadata(router.app.AppFs.Fs, course2.Path)
		require.NoError(t, err)
		require.NotNil(t, metadata2)
		require.Equal(t, []string{"Python"}, metadata2.Tags)

		// Verify tag was deleted from DB
		count, err := router.appDao.CountTags(ctx, nil)
		require.NoError(t, err)
		require.Equal(t, 2, count) // Only PHP and Python remain
	})
}
