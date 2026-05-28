package dao

import (
	"database/sql"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/types"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_CreateAssetKeyframes(t *testing.T) {
	// Test successfully inserting an asset keyframes record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     "/course-1/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		keyframes := &models.AssetKeyframes{
			AssetID:    asset.ID,
			Keyframes:  []float64{0.0, 2.5, 5.0, 7.5, 10.0},
			IsComplete: true,
		}

		err := dao.CreateAssetKeyframes(ctx, keyframes)
		require.NoError(t, err)
		require.NotEmpty(t, keyframes.ID)
		require.NotEmpty(t, keyframes.CreatedAt)
		require.NotEmpty(t, keyframes.UpdatedAt)
	})

	// Test error due to nil keyframes pointer
	t.Run("nil keyframes", func(t *testing.T) {
		dao, ctx := setup(t)

		require.ErrorIs(t, dao.CreateAssetKeyframes(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to empty asset ID
	t.Run("empty asset ID", func(t *testing.T) {
		dao, ctx := setup(t)

		keyframes := &models.AssetKeyframes{
			AssetID: "",
		}

		err := dao.CreateAssetKeyframes(ctx, keyframes)
		require.Error(t, err)
		require.ErrorIs(t, err, utils.ErrAssetId)
	})

	// Test error due to out-of-order keyframes
	t.Run("invalid keyframes", func(t *testing.T) {
		dao, ctx := setup(t)

		keyframes := &models.AssetKeyframes{
			AssetID:   "any-asset-id",
			Keyframes: []float64{5.0, 2.5, 10.0},
		}

		err := dao.CreateAssetKeyframes(ctx, keyframes)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not in ascending order")
	})

	// Test error due to duplicate asset ID
	t.Run("duplicate asset ID", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     "/course-1/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		keyframes1 := &models.AssetKeyframes{
			AssetID:    asset.ID,
			Keyframes:  []float64{0.0, 2.5},
			IsComplete: false,
		}
		require.NoError(t, dao.CreateAssetKeyframes(ctx, keyframes1))

		keyframes2 := &models.AssetKeyframes{
			AssetID:    asset.ID,
			Keyframes:  []float64{0.0, 2.5, 5.0},
			IsComplete: true,
		}
		err := dao.CreateAssetKeyframes(ctx, keyframes2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "UNIQUE constraint failed")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetAssetKeyframes(t *testing.T) {
	// Test successfully retrieving an asset keyframes record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     "/course-1/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		keyframes := &models.AssetKeyframes{
			AssetID:    asset.ID,
			Keyframes:  []float64{0.0, 2.5, 5.0, 7.5, 10.0},
			IsComplete: true,
		}
		require.NoError(t, dao.CreateAssetKeyframes(ctx, keyframes))

		retrieved, err := dao.GetAssetKeyframes(ctx, asset.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)
		require.Equal(t, keyframes.ID, retrieved.ID)
		require.Equal(t, asset.ID, retrieved.AssetID)
		require.Equal(t, types.Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}, retrieved.Keyframes)
		require.True(t, retrieved.IsComplete)
		require.NotEmpty(t, retrieved.CreatedAt)
		require.NotEmpty(t, retrieved.UpdatedAt)
	})

	// Test querying a non-existent asset keyframes record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		retrieved, err := dao.GetAssetKeyframes(ctx, "non-existent-asset")
		require.NoError(t, err)
		require.Nil(t, retrieved)
	})

	// Test error due to empty asset ID
	t.Run("empty asset ID", func(t *testing.T) {
		dao, ctx := setup(t)

		retrieved, err := dao.GetAssetKeyframes(ctx, "")
		require.Error(t, err)
		require.ErrorIs(t, err, utils.ErrAssetId)
		require.Nil(t, retrieved)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteAssetKeyframes(t *testing.T) {
	// Test cascading delete of asset keyframes records
	t.Run("cascade", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     "/course-1/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		keyframes := &models.AssetKeyframes{
			AssetID:    asset.ID,
			Keyframes:  []float64{0.0, 2.5, 5.0, 7.5, 10.0},
			IsComplete: true,
		}

		require.NoError(t, dao.CreateAssetKeyframes(ctx, keyframes))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{"id": asset.ID})
		require.NoError(t, dao.DeleteAssets(ctx, dbOpts))

		record, err := dao.GetAssetKeyframes(ctx, asset.ID)
		require.NoError(t, err)
		require.Nil(t, record)

	})
}
