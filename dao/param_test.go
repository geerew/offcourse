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

func Test_CreateParam(t *testing.T) {
	// Test successfully inserting a param record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, param))
	})

	// Test error due to duplicate record
	t.Run("duplicate", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, param))

		require.ErrorContains(t, dao.CreateParam(ctx, param), "UNIQUE constraint failed: "+models.PARAM_TABLE_KEY)
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)

		require.ErrorIs(t, dao.CreateParam(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to invalid key
	t.Run("invalid key", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "", Value: "value 1"}
		require.ErrorIs(t, dao.CreateParam(ctx, param), utils.ErrKey)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetParam(t *testing.T) {
	// Test successfully retrieving a param record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, param))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.PARAM_TABLE_KEY: param.Key})
		record, err := dao.GetParam(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, param.ID, record.ID)
	})

	// Test no error when retrieving a non-existent param record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetParam(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListParams(t *testing.T) {
	// Test successfully retrieving all param records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		params := []*models.Param{}

		for i := range 3 {
			param := &models.Param{Key: fmt.Sprintf("param %d", i), Value: fmt.Sprintf("value %d", i)}
			params = append(params, param)
			require.NoError(t, dao.CreateParam(ctx, param))
			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListParams(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, params[i].ID, record.ID)
		}
	})

	// Test successfully retrieving no param records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListParams(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered param records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		params := []*models.Param{}
		for i := range 3 {
			param := &models.Param{Key: fmt.Sprintf("param %d", i), Value: fmt.Sprintf("value %d", i)}
			params = append(params, param)
			require.NoError(t, dao.CreateParam(ctx, param))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.PARAM_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListParams(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, params[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.PARAM_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListParams(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, params[i].ID, record.ID)
		}
	})

	// Test successfully retrieving selected param records
	t.Run("where", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, param))

		opts := NewOptions().WithWhere(squirrel.Eq{models.PARAM_TABLE_ID: param.ID})
		records, err := dao.ListParams(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, param.ID, records[0].ID)
	})

	// Test successfully retrieving paginated param records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		params := []*models.Param{}
		for i := range 17 {
			param := &models.Param{Key: fmt.Sprintf("param %d", i), Value: fmt.Sprintf("value %d", i)}
			params = append(params, param)
			require.NoError(t, dao.CreateParam(ctx, param))
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().WithPagination(pagination.New(1, 10))
		records, err := dao.ListParams(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, params[0].ID, records[0].ID)
		require.Equal(t, params[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().WithPagination(pagination.New(2, 10))
		records, err = dao.ListParams(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, params[10].ID, records[0].ID)
		require.Equal(t, params[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpdateParam(t *testing.T) {
	// Test successfully updating a param record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		originalParam := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, originalParam))

		time.Sleep(1 * time.Millisecond)

		updatedParam := &models.Param{
			Base:  originalParam.Base,
			Key:   "param 1",   // Immutable
			Value: "new value", // Mutable
		}
		require.NoError(t, dao.UpdateParam(ctx, updatedParam))

		time.Sleep(1 * time.Millisecond)
		require.NoError(t, dao.UpdateParam(ctx, updatedParam))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.PARAM_TABLE_ID: originalParam.ID})
		record, err := dao.GetParam(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, originalParam.ID, record.ID)                     // No change
		require.True(t, record.CreatedAt.Equal(originalParam.CreatedAt))  // No change
		require.Equal(t, updatedParam.Value, record.Value)                // Changed
		require.False(t, record.UpdatedAt.Equal(originalParam.UpdatedAt)) // Changed
	})

	// Test error due to invalid param record
	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, param))

		// Empty Key
		param.Key = ""
		require.ErrorIs(t, dao.UpdateParam(ctx, param), utils.ErrKey)

		// Empty ID
		param.ID = ""
		require.ErrorIs(t, dao.UpdateParam(ctx, param), utils.ErrId)

		// Nil Model
		require.ErrorIs(t, dao.UpdateParam(ctx, nil), utils.ErrNilPtr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteParams(t *testing.T) {
	// Test successfully deleting a param record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, param))

		opts := NewOptions().WithWhere(squirrel.Eq{models.PARAM_TABLE_ID: param.ID})
		require.Nil(t, dao.DeleteParams(ctx, opts))

		records, err := dao.ListParams(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent param record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, param))

		opts := NewOptions().WithWhere(squirrel.Eq{models.PARAM_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteParams(ctx, opts))

		records, err := dao.ListParams(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, param.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		param := &models.Param{Key: "param 1", Value: "value 1"}
		require.NoError(t, dao.CreateParam(ctx, param))

		require.ErrorIs(t, dao.DeleteParams(ctx, nil), utils.ErrWhere)

		records, err := dao.ListParams(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, param.ID, records[0].ID)
	})
}
