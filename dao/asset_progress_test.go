package dao

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpsertAssetProgress(t *testing.T) {
	// Test successfully creating then updating an asset progress record when video metadata
	// is available
	t.Run("success (with video metadata)", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "L1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "M1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Video A",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "M1",
			Type:     types.MustAsset("mp4"),
			Path:     "/course-1/01-video-a.mp4",
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "hash-a",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		// Set video duration of 100 seconds
		meta := &models.AssetMetadata{
			AssetID: asset.ID,
			VideoMetadata: &models.VideoMetadata{
				DurationSec: 100,
				Container:   "mp4",
				MIMEType:    "video/mp4",
				VideoCodec:  "h264",
				Width:       1280,
				Height:      720,
				FPSNum:      30,
				FPSDen:      1,
			},
		}
		require.NoError(t, dao.CreateAssetMetadata(ctx, meta))

		// Create with an initial video position of 50 seconds
		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 50,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_PROGRESS_TABLE_ID: assetProgress.ID})
		record, err := dao.GetAssetProgress(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, asset.ID, record.AssetID)
		require.Equal(t, 50, record.Position)
		require.InEpsilon(t, 0.5, record.ProgressFrac, 1e-9)
		require.False(t, record.Completed)
		require.True(t, record.CompletedAt.IsZero())

		// Update video position to 100 and mark completed
		assetProgress.Position = 100
		assetProgress.Completed = true
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		record, err = dao.GetAssetProgress(ctx, opts)
		require.NoError(t, err)
		require.Equal(t, 100, record.Position)
		require.InEpsilon(t, 1.0, record.ProgressFrac, 1e-9)
		require.True(t, record.Completed)
		require.False(t, record.CompletedAt.IsZero())
	})

	// Test successfully creating then updating an asset progress record when video metadata
	// is not available
	t.Run("success (without video metadata)", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 2", Path: "/course-2"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "L2",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Video B (no metadata)",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Type:     types.MustAsset("mp4"),
			Path:     "/course-2/01-video-b.mp4",
			FileSize: 2048,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "hash-b",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		// Create with an initial video position of 50 seconds
		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 50,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_PROGRESS_TABLE_ID: assetProgress.ID})
		record, err := dao.GetAssetProgress(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, 50, record.Position)
		require.InDelta(t, 0.0, record.ProgressFrac, 1e-9)
		require.False(t, record.Completed)
		require.True(t, record.CompletedAt.IsZero())

		// Mark completed
		assetProgress.Completed = true
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		record, err = dao.GetAssetProgress(ctx, opts)
		require.NoError(t, err)
		require.True(t, record.Completed)
		require.InDelta(t, 1.0, record.ProgressFrac, 1e-9)
	})

	// Test successfully creating and updating multiple asset progress records
	t.Run("success (multiple assets)", func(t *testing.T) {
		dao, ctx := setup(t)

		// Course + lesson
		course := &models.Course{Title: "Course 3", Path: "/course-3"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "L3",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		// Video #1 (video duration 120)
		video1 := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Video 120s",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Type:     types.MustAsset("mp4"),
			Path:     "/course-3/01-120s.mp4",
			FileSize: 1111,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "v1",
		}
		require.NoError(t, dao.CreateAsset(ctx, video1))

		require.NoError(t, dao.CreateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID: video1.ID,
			VideoMetadata: &models.VideoMetadata{
				DurationSec: 120,
				MIMEType:    "video/mp4",
				Container:   "mp4",
				VideoCodec:  "h264",
				Width:       1280,
				Height:      720,
				FPSNum:      30,
				FPSDen:      1,
			},
		}))

		// Video #2 (video duration 30)
		video2 := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Video 30s",
			Prefix:   sql.NullInt16{Int16: 2, Valid: true},
			Type:     types.MustAsset("mp4"),
			Path:     "/course-3/02-30s.mp4",
			FileSize: 2222,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "v2",
		}
		require.NoError(t, dao.CreateAsset(ctx, video2))

		// Document #1 (no metadata)
		doc := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Doc",
			Prefix:   sql.NullInt16{Int16: 3, Valid: true},
			Type:     types.MustAsset("md"),
			Path:     "/course-3/03-doc.md",
			FileSize: 3333,
			ModTime:  time.Now().Format(time.RFC3339Nano), Hash: "d1",
		}
		require.NoError(t, dao.CreateAsset(ctx, doc))

		// Create video1 with a position of 60 seconds (50%)
		video1Progress := &models.AssetProgress{AssetID: video1.ID, Position: 60}
		require.NoError(t, dao.UpsertAssetProgress(ctx, video1Progress))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_PROGRESS_TABLE_ID: video1Progress.ID})
		record, err := dao.GetAssetProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.InDelta(t, 0.5, record.ProgressFrac, 1e-9)

		// Create video2 with a position of 15 seconds (50%)
		video2Progress := &models.AssetProgress{AssetID: video2.ID, Position: 15}
		require.NoError(t, dao.UpsertAssetProgress(ctx, video2Progress))

		record, err = dao.GetAssetProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.InDelta(t, 0.5, record.ProgressFrac, 1e-9)

		// Create document marked as completed
		documentProgress := &models.AssetProgress{AssetID: doc.ID, Completed: true}
		require.NoError(t, dao.UpsertAssetProgress(ctx, documentProgress))

		docDbOpts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_PROGRESS_TABLE_ID: documentProgress.ID})
		record, err = dao.GetAssetProgress(ctx, docDbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.True(t, record.Completed)
		require.InDelta(t, 1.0, record.ProgressFrac, 1e-9)
	})

	// Test error due to nil pointer
	t.Run("nil", func(t *testing.T) {
		dao, ctx := setup(t)
		require.ErrorIs(t, dao.UpsertAssetProgress(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to missing principal
	t.Run("missing principal", func(t *testing.T) {
		dao, _ := setup(t)
		require.ErrorIs(t, dao.UpsertAssetProgress(context.Background(), &models.AssetProgress{AssetID: "1234"}), utils.ErrPrincipal)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetAssetProgress(t *testing.T) {
	// Test successfully retrieving an asset progress record
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
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "1234",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 50,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_PROGRESS_TABLE_ID: assetProgress.ID})
		record, err := dao.GetAssetProgress(ctx, opts)
		require.NoError(t, err)
		require.Equal(t, assetProgress.ID, record.ID)
	})

	// Test querying a non-existent asset progress record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetAssetProgress(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListAssetProgress(t *testing.T) {
	// Test successfully retrieving all asset progress records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		assetProgresses := []*models.AssetProgress{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
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
				Path:     fmt.Sprintf("/course-%d/01 asset.mp4", i),
				FileSize: 1024,
				ModTime:  time.Now().Format(time.RFC3339Nano),
				Hash:     "1234",
			}
			require.NoError(t, dao.CreateAsset(ctx, asset))

			// Set the asset progress to 5
			assetProgress := &models.AssetProgress{
				AssetID:  asset.ID,
				Position: 5,
			}
			assetProgresses = append(assetProgresses, assetProgress)
			require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))
			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListAssetProgress(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, assetProgresses[i].ID, record.ID)
		}
	})

	// Test successfully retrieving no asset progress records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListAssetProgress(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered asset progress records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		assetProgresses := []*models.AssetProgress{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
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
				Path:     fmt.Sprintf("/course-%d/01 asset.mp4", i),
				FileSize: 1024,
				ModTime:  time.Now().Format(time.RFC3339Nano),
				Hash:     "1234",
			}
			require.NoError(t, dao.CreateAsset(ctx, asset))

			// Set the asset progress to 5
			assetProgress := &models.AssetProgress{
				AssetID:  asset.ID,
				Position: 5,
			}
			assetProgresses = append(assetProgresses, assetProgress)
			require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.ASSET_PROGRESS_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListAssetProgress(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, assetProgresses[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.ASSET_PROGRESS_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListAssetProgress(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, assetProgresses[i].ID, record.ID)
		}
	})

	// Test successfully retrieving selected asset progress records
	t.Run("where", func(t *testing.T) {
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
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "1234",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 50,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_PROGRESS_TABLE_ID: assetProgress.ID})
		records, err := dao.ListAssetProgress(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, assetProgress.ID, records[0].ID)
	})

	// Test successfully retrieving paginated asset progress records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		assetProgresses := []*models.AssetProgress{}
		for i := range 17 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
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
				Path:     fmt.Sprintf("/course-%d/01 asset.mp4", i),
				FileSize: 1024,
				ModTime:  time.Now().Format(time.RFC3339Nano),
				Hash:     "1234",
			}
			require.NoError(t, dao.CreateAsset(ctx, asset))

			// Set the asset progress to 5
			assetProgress := &models.AssetProgress{
				AssetID:  asset.ID,
				Position: 5,
			}
			assetProgresses = append(assetProgresses, assetProgress)
			require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().WithPagination(pagination.New(1, 10))
		records, err := dao.ListAssetProgress(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, assetProgresses[0].ID, records[0].ID)
		require.Equal(t, assetProgresses[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().WithPagination(pagination.New(2, 10))
		records, err = dao.ListAssetProgress(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, assetProgresses[10].ID, records[0].ID)
		require.Equal(t, assetProgresses[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteAssetProgress(t *testing.T) {
	// Test successfully deleting an asset progress record
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
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "1234",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 50,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_PROGRESS_TABLE_ID: assetProgress.ID})
		require.Nil(t, dao.DeleteAssetProgress(ctx, opts))

		records, err := dao.ListAssetProgress(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent asset progress record
	t.Run("not found", func(t *testing.T) {
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
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "1234",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 50,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_PROGRESS_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteAssetProgress(ctx, opts))

		records, err := dao.ListAssetProgress(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, assetProgress.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
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
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "1234",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 50,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		require.ErrorIs(t, dao.DeleteAssetProgress(ctx, nil), utils.ErrWhere)

		records, err := dao.ListAssetProgress(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, assetProgress.ID, records[0].ID)
	})

	// Test cascading delete of asset progress records when deleting a course
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
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "1234",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 50,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		require.Nil(t, dao.DeleteCourses(ctx, dbOpts))

		records, err := dao.ListAssetProgress(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)
	})
}
