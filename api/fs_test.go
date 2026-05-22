package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/types"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestFsPath(t *testing.T) {
	t.Run("200 (found)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		router.app.FS.MkdirAll("/dir1", os.ModePerm)
		router.app.FS.Create("/file1")
		router.app.FS.Create("/file2")
		router.app.FS.Create("/file3")

		req := httptest.NewRequest(http.MethodGet, "/api/filesystem/"+utils.EncodeString("/"), nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var respData fileSystemResponse
		err = json.Unmarshal(body, &respData)
		require.NoError(t, err)
		require.Equal(t, 4, respData.Count)
		require.Len(t, respData.Directories, 1)
		require.Len(t, respData.Files, 3)
		require.Equal(t, types.PathClassificationNone, respData.Directories[0].Classification)
	})

	t.Run("200 (path classifications)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		// Create /dir 1, /course 1, /courses/course 2
		router.app.FS.MkdirAll("/dir 1", os.ModePerm)

		course1 := &models.Course{Title: "course 1", Path: "/course 1"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course1))
		require.NoError(t, router.app.FS.MkdirAll(course1.Path, os.ModePerm))

		course2 := &models.Course{Title: "course 2", Path: "/courses/course 2"}
		require.NoError(t, router.appDao.CreateCourse(ctx, course2))
		require.NoError(t, router.app.FS.MkdirAll(course2.Path, os.ModePerm))

		// Test /
		req := httptest.NewRequest(http.MethodGet, "/api/filesystem/"+utils.EncodeString("/"), nil)
		status, body, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		var respData fileSystemResponse
		err = json.Unmarshal(body, &respData)
		require.NoError(t, err)
		require.Equal(t, 3, respData.Count)
		require.Len(t, respData.Directories, 3)
		require.Len(t, respData.Files, 0)

		require.Equal(t, types.PathClassificationCourse, respData.Directories[0].Classification)   // /course 1
		require.Equal(t, types.PathClassificationAncestor, respData.Directories[1].Classification) // /courses
		require.Equal(t, types.PathClassificationNone, respData.Directories[2].Classification)     // /dir 1
	})

	t.Run("404 (path not found)", func(t *testing.T) {
		router, _ := setupAdmin(t)
		req := httptest.NewRequest(http.MethodGet, "/api/filesystem/"+utils.EncodeString("/other"), nil)
		status, _, err := requestHelper(t, router, req)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, status)
	})

	t.Run("400 (decode error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		req := httptest.NewRequest(http.MethodGet, "/api/filesystem/`", nil)
		status, body, err := requestHelper(t, router, req)

		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, status)

		var respData map[string]string
		err = json.Unmarshal(body, &respData)
		require.NoError(t, err)
		require.Contains(t, respData["message"], "Invalid path encoding")
	})
}
