package dao

import (
	"fmt"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_CreateLog(t *testing.T) {
	// Test successfully inserting a log record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setupLog(t)

		log := &models.Log{Data: map[string]any{}, Level: "info", Message: fmt.Sprintf("log %d", 1)}
		require.NoError(t, dao.CreateLog(ctx, log))
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setupLog(t)

		require.ErrorIs(t, dao.CreateLog(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to invalid message
	t.Run("invalid message", func(t *testing.T) {
		dao, ctx := setupLog(t)

		log := &models.Log{Data: map[string]any{}, Level: "info", Message: ""}
		require.ErrorIs(t, dao.CreateLog(ctx, log), utils.ErrLogMessage)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetLog(t *testing.T) {
	// Test successfully retrieving a log record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setupLog(t)

		log := &models.Log{Data: map[string]any{}, Level: "info", Message: fmt.Sprintf("log %d", 1)}
		require.NoError(t, dao.CreateLog(ctx, log))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.LOG_TABLE_ID: log.ID})
		record, err := dao.GetLog(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, log.ID, record.ID)
	})

	// Test no error when retrieving a non-existent log record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setupLog(t)

		record, err := dao.GetLog(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListLogs(t *testing.T) {
	// Test successfully retrieving all log records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setupLog(t)

		logs := []*models.Log{}

		for i := range 3 {
			log := &models.Log{Data: map[string]any{}, Level: "info", Message: fmt.Sprintf("log %d", i+1)}
			logs = append(logs, log)
			require.NoError(t, dao.CreateLog(ctx, log))
			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListLogs(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, logs[i].ID, record.ID)
		}
	})

	// Test successfully retrieving no log records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setupLog(t)

		records, err := dao.ListLogs(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered log records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setupLog(t)

		logs := []*models.Log{}
		for i := range 3 {
			log := &models.Log{Data: map[string]any{}, Level: "info", Message: fmt.Sprintf("log %d", i+1)}
			logs = append(logs, log)
			require.NoError(t, dao.CreateLog(ctx, log))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.LOG_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListLogs(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, logs[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.LOG_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListLogs(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, logs[i].ID, record.ID)
		}
	})

	// Test successfully retrieving selected log records
	t.Run("where", func(t *testing.T) {
		dao, ctx := setupLog(t)

		log := &models.Log{Data: map[string]any{}, Level: "info", Message: fmt.Sprintf("log %d", 1)}
		require.NoError(t, dao.CreateLog(ctx, log))

		opts := NewOptions().WithWhere(squirrel.Eq{models.LOG_TABLE_ID: log.ID})
		records, err := dao.ListLogs(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, log.ID, records[0].ID)
	})

	// Test successfully retrieving paginated log records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setupLog(t)

		logs := []*models.Log{}
		for i := range 17 {
			log := &models.Log{Data: map[string]any{}, Level: "info", Message: fmt.Sprintf("log %d", i+1)}
			logs = append(logs, log)
			require.NoError(t, dao.CreateLog(ctx, log))
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().
			WithOrderBy(models.LOG_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(1, 10))

		records, err := dao.ListLogs(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, logs[0].ID, records[0].ID)
		require.Equal(t, logs[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().
			WithOrderBy(models.LOG_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(2, 10))

		records, err = dao.ListLogs(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, logs[10].ID, records[0].ID)
		require.Equal(t, logs[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_CreateLogsBatch(t *testing.T) {
	// Test successfully inserting multiple log records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setupLog(t)

		logs := []*models.Log{}
		levels := []string{"debug", "info", "warn"}
		for i := range 5 {
			log := &models.Log{
				Data:    map[string]any{"test": i},
				Level:   levels[i%len(levels)],
				Message: fmt.Sprintf("batch log %d", i+1),
			}
			logs = append(logs, log)
		}

		require.NoError(t, dao.CreateLogsBatch(ctx, logs))

		// Verify all logs were inserted
		records, err := dao.ListLogs(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 5)

		// Check that all messages are present
		messages := make(map[string]bool)
		for _, record := range records {
			messages[record.Message] = true
		}

		for _, log := range logs {
			require.True(t, messages[log.Message], "Message %s should be present", log.Message)
		}
	})

	// Test successfully inserting no log records
	t.Run("empty slice", func(t *testing.T) {
		dao, ctx := setupLog(t)

		logs := []*models.Log{}
		require.NoError(t, dao.CreateLogsBatch(ctx, logs))

		// Verify no logs were inserted
		records, err := dao.ListLogs(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test error due to nil pointer in slice
	t.Run("nil pointer in slice", func(t *testing.T) {
		dao, ctx := setupLog(t)

		logs := []*models.Log{
			{Data: map[string]any{}, Level: "info", Message: "valid log"},
			nil,
			{Data: map[string]any{}, Level: "error", Message: "another valid log"},
		}

		require.ErrorIs(t, dao.CreateLogsBatch(ctx, logs), utils.ErrNilPtr)
	})

	// Test error due to invalid message in slice
	t.Run("invalid message in slice", func(t *testing.T) {
		dao, ctx := setupLog(t)

		logs := []*models.Log{
			{Data: map[string]any{}, Level: "info", Message: "valid log"},
			{Data: map[string]any{}, Level: "info", Message: ""}, // Invalid
			{Data: map[string]any{}, Level: "error", Message: "another valid log"},
		}

		require.ErrorIs(t, dao.CreateLogsBatch(ctx, logs), utils.ErrLogMessage)
	})

	// Test successfully inserting a large number of log records
	t.Run("large batch", func(t *testing.T) {
		dao, ctx := setupLog(t)

		logs := []*models.Log{}
		levels := []string{"debug", "info", "warn"}
		for i := range 100 {
			log := &models.Log{
				Data:    map[string]any{"index": i},
				Level:   levels[i%len(levels)],
				Message: fmt.Sprintf("large batch log %d", i+1),
			}
			logs = append(logs, log)
		}

		require.NoError(t, dao.CreateLogsBatch(ctx, logs))

		// Verify all logs were inserted
		records, err := dao.ListLogs(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 100)
	})
}
