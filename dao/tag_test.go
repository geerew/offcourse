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

func Test_CreateTag(t *testing.T) {
	// Test successfully creating a tag record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, tag))
	})

	// Test error due to duplicate record
	t.Run("duplicate", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, tag))

		// Attempt to create a duplicate tag
		duplicateTag := &models.Tag{Tag: "Tag 1"}
		require.ErrorContains(t, dao.CreateTag(ctx, duplicateTag), "UNIQUE constraint failed: "+models.TAG_TABLE_TAG)

		// Attempt to create a duplicate tag with different case
		duplicateTag = &models.Tag{Tag: "tag 1"}
		require.ErrorContains(t, dao.CreateTag(ctx, duplicateTag), "UNIQUE constraint failed: "+models.TAG_TABLE_TAG)
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)

		require.ErrorIs(t, dao.CreateTag(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to invalid tag
	t.Run("invalid tag", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: ""}
		require.ErrorIs(t, dao.CreateTag(ctx, tag), utils.ErrTag)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TODO Add test to check course count aggregation
func Test_GetTag(t *testing.T) {
	// Test successfully retrieving a tag record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, tag))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: tag.ID})
		record, err := dao.GetTag(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, tag.ID, record.ID)
		require.Equal(t, tag.Tag, record.Tag)
		require.Zero(t, record.CourseCount)
	})

	// Test no error when retrieving a non-existent tag record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetTag(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListTags(t *testing.T) {
	// Test successfully retrieving all tag records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		tags := []*models.Tag{}
		for i := range 3 {
			tag := &models.Tag{Tag: fmt.Sprintf("Tag %d", i)}
			tags = append(tags, tag)
			require.NoError(t, dao.CreateTag(ctx, tag))
			time.Sleep(1 * time.Millisecond)
		}

		dbOpts := NewOptions().WithOrderBy(models.TAG_TABLE_CREATED_AT + " ASC")
		records, err := dao.ListTags(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, tags[i].ID, record.ID)
		}
	})

	// Test successfully retrieving no tag records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListTags(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered tag records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		tags := []*models.Tag{}
		for i := range 3 {
			tag := &models.Tag{Tag: fmt.Sprintf("Tag %d", i)}
			tags = append(tags, tag)
			require.NoError(t, dao.CreateTag(ctx, tag))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.TAG_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListTags(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, tags[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.TAG_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListTags(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, tags[i].ID, record.ID)
		}
	})

	// Test successfully retrieving selected tag records
	t.Run("where", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, tag))

		opts := NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: tag.ID})
		records, err := dao.ListTags(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, tag.ID, records[0].ID)
	})

	// Test successfully retrieving paginated tag records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		tags := []*models.Tag{}
		for i := range 17 {
			tag := &models.Tag{Tag: fmt.Sprintf("Tag %d", i)}
			require.NoError(t, dao.CreateTag(ctx, tag))
			tags = append(tags, tag)

			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		dbOpts := NewOptions().
			WithOrderBy(models.TAG_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(1, 10))
		records, err := dao.ListTags(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, tags[0].ID, records[0].ID)
		require.Equal(t, tags[9].ID, records[9].ID)

		// Second page with remaining 7 records
		dbOpts = NewOptions().
			WithOrderBy(models.TAG_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(2, 10))
		records, err = dao.ListTags(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, tags[10].ID, records[0].ID)
		require.Equal(t, tags[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpdateTag(t *testing.T) {
	// Test successfully updating a tag record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		originalTag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, originalTag))

		updatedTag := &models.Tag{
			Base: originalTag.Base,
			Tag:  "Tag 2", // Mutable
		}

		time.Sleep(1 * time.Millisecond)
		require.NoError(t, dao.UpdateTag(ctx, updatedTag))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: originalTag.ID})
		record, err := dao.GetTag(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, originalTag.ID, record.ID)                     // No change
		require.Equal(t, updatedTag.Tag, record.Tag)                    // Changed
		require.False(t, record.UpdatedAt.Equal(originalTag.UpdatedAt)) // Changed
	})

	// Test error due to invalid tag
	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, tag))

		// Empty tag
		tag.Tag = ""
		require.ErrorIs(t, dao.UpdateTag(ctx, tag), utils.ErrTag)

		// Empty ID
		tag.ID = ""
		require.ErrorIs(t, dao.UpdateTag(ctx, tag), utils.ErrId)

		// Nil Model
		require.ErrorIs(t, dao.UpdateTag(ctx, nil), utils.ErrNilPtr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteTags(t *testing.T) {
	// Test successfully deleting a tag record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, tag))

		opts := NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: tag.ID})
		require.Nil(t, dao.DeleteTags(ctx, opts))

		records, err := dao.ListTags(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent tag record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, tag))

		opts := NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteTags(ctx, opts))

		records, err := dao.ListTags(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, tag.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Tag 1"}
		require.NoError(t, dao.CreateTag(ctx, tag))

		require.ErrorIs(t, dao.DeleteTags(ctx, nil), utils.ErrWhere)

		records, err := dao.ListTags(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, tag.ID, records[0].ID)
	})
}
