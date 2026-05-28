package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogs_GetLogs(t *testing.T) {
	t.Run("200 (empty)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/logs/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, _ := unmarshalHelper[logResponse](t, body)
		require.Zero(t, int(paginationResp.TotalItems))
		require.Zero(t, len(paginationResp.Items))
	})

	t.Run("200 (found)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		for i := range 5 {
			log := &models.Log{
				Data:    map[string]any{},
				Level:   "info",
				Message: fmt.Sprintf("log %d", i+1),
			}

			require.Nil(t, router.logDao.CreateLog(ctx, log))
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/logs/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, logResponses := unmarshalHelper[logResponse](t, body)
		require.Equal(t, 5, int(paginationResp.TotalItems))
		require.Len(t, logResponses, 5)
	})

	t.Run("200 (pagination)", func(t *testing.T) {
		router, ctx := setupAdmin(t)

		for i := range 17 {
			require.Nil(t, router.logDao.CreateLog(ctx, &models.Log{Data: map[string]any{}, Level: "info", Message: fmt.Sprintf("log %d", i+1)}))
			time.Sleep(1 * time.Millisecond)
		}

		// Get the first page (10 logs)
		params := url.Values{
			pagination.PageQueryParam:    {"1"},
			pagination.PerPageQueryParam: {"10"},
		}

		status, body, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/logs/?"+params.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, logResponses := unmarshalHelper[logResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 10)
		require.Equal(t, "log 17", logResponses[0].Message)
		require.Equal(t, "log 8", logResponses[9].Message)

		// Get the second page (7 logs)
		params = url.Values{
			pagination.PageQueryParam:    {"2"},
			pagination.PerPageQueryParam: {"10"},
		}
		status, body, err = requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/logs/?"+params.Encode(), nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, status)

		paginationResp, logResponses = unmarshalHelper[logResponse](t, body)
		require.Equal(t, 17, int(paginationResp.TotalItems))
		require.Len(t, paginationResp.Items, 7)
		require.Equal(t, "log 7", logResponses[0].Message)
		require.Equal(t, "log 1", logResponses[6].Message)
	})

	t.Run("500 (internal error)", func(t *testing.T) {
		router, _ := setupAdmin(t)

		// Drop the courses table
		_, err := router.app.DbManager.LogsDb.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.LOG_TABLE)
		require.NoError(t, err)

		status, _, err := requestHelper(t, router, httptest.NewRequest(http.MethodGet, "/api/logs/", nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusInternalServerError, status)
	})
}
