package dao

import (
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

func Test_GetCourseProgress(t *testing.T) {
	// Test successfully retrieving course progress with a single asset
	t.Run("success (video)", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))
		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Lesson 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		video := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Video 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Type:     types.MustAsset("mp4"),
			Path:     "/course-1/01-video.mp4",
			FileSize: 111,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "v1",
			Weight:   1,
		}
		require.NoError(t, dao.CreateAsset(ctx, video))

		require.NoError(t, dao.CreateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID: video.ID,
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
		}))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_PROGRESS_TABLE_COURSE_ID: course.ID})
		record, err := dao.GetCourseProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.Nil(t, record)

		ap := &models.AssetProgress{AssetID: video.ID, Position: 50}
		require.NoError(t, dao.UpsertAssetProgress(ctx, ap))

		record, err = dao.GetCourseProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, course.ID, record.CourseID)
		require.True(t, record.Started)
		require.False(t, record.StartedAt.IsZero())
		require.Equal(t, 50, record.Percent)
		require.True(t, record.CompletedAt.IsZero())

		ap.Completed = true
		ap.Position = 100
		require.NoError(t, dao.UpsertAssetProgress(ctx, ap))

		record, err = dao.GetCourseProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, 100, record.Percent)
		require.False(t, record.CompletedAt.IsZero())
		require.False(t, record.StartedAt.IsZero())
	})

	// Test successfully retrieving course progress with multiple assets
	t.Run("success (multiple assets)", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course M", Path: "/course-m"}
		require.NoError(t, dao.CreateCourse(ctx, course))
		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Lesson M",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		video1 := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Video A",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Type:     types.MustAsset("mp4"),
			Path:     "/course-m/01-a.mp4",
			FileSize: 100,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "va",
			Weight:   1,
		}
		require.NoError(t, dao.CreateAsset(ctx, video1))

		require.NoError(t, dao.CreateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID: video1.ID,
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
		}))

		video2 := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Video B",
			Prefix:   sql.NullInt16{Int16: 2, Valid: true},
			Type:     types.MustAsset("mp4"),
			Path:     "/course-m/02-b.mp4",
			FileSize: 200,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "vb",
			Weight:   1,
		}
		require.NoError(t, dao.CreateAsset(ctx, video2))

		require.NoError(t, dao.CreateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID: video2.ID,
			VideoMetadata: &models.VideoMetadata{
				DurationSec: 200,
				Container:   "mp4",
				MIMEType:    "video/mp4",
				VideoCodec:  "h264",
				Width:       1920,
				Height:      1080,
				FPSNum:      30,
				FPSDen:      1,
			},
		}))

		doc := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Doc C",
			Prefix:   sql.NullInt16{Int16: 3, Valid: true},
			Type:     types.MustAsset("md"),
			Path:     "/course-m/03-c.md",
			FileSize: 10,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "dc",
			Weight:   1,
		}
		require.NoError(t, dao.CreateAsset(ctx, doc))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_PROGRESS_TABLE_COURSE_ID: course.ID})

		video1Progress := &models.AssetProgress{AssetID: video1.ID, Position: 50}
		require.NoError(t, dao.UpsertAssetProgress(ctx, video1Progress))

		record, err := dao.GetCourseProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, 17, record.Percent)
		require.True(t, record.Started)
		require.False(t, record.StartedAt.IsZero())

		video2Progress := &models.AssetProgress{AssetID: video2.ID, Position: 100}
		require.NoError(t, dao.UpsertAssetProgress(ctx, video2Progress))

		record, err = dao.GetCourseProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, 33, record.Percent)

		documentProgress := &models.AssetProgress{AssetID: doc.ID, Completed: true}
		require.NoError(t, dao.UpsertAssetProgress(ctx, documentProgress))

		record, err = dao.GetCourseProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, 67, record.Percent)
		require.True(t, record.CompletedAt.IsZero())

		video1Progress.Completed = true
		video1Progress.Position = 100
		require.NoError(t, dao.UpsertAssetProgress(ctx, video1Progress))

		video2Progress.Completed = true
		video2Progress.Position = 200
		require.NoError(t, dao.UpsertAssetProgress(ctx, video2Progress))

		record, err = dao.GetCourseProgress(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, 100, record.Percent)
		require.False(t, record.CompletedAt.IsZero())
	})

	// Test querying a non-existent course progress record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetCourseProgress(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListCourseProgress(t *testing.T) {
	// Test successfully retrieving all course progress records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}

		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			courses = append(courses, course)
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

			assetProgress := &models.AssetProgress{
				AssetID:  asset.ID,
				Position: 5,
			}
			require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))
			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListCourseProgress(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courses[i].ID, record.CourseID)
		}
	})

	// Test successfully retrieving no course progress records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListCourseProgress(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered course progress records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			courses = append(courses, course)
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

			assetProgress := &models.AssetProgress{
				AssetID:  asset.ID,
				Position: 5,
			}
			require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.COURSE_PROGRESS_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListCourseProgress(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courses[2-i].ID, record.CourseID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.COURSE_PROGRESS_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListCourseProgress(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courses[i].ID, record.CourseID)
		}
	})

	// Test successfully retrieving selected course progress records
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
			Position: 5,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_PROGRESS_TABLE_COURSE_ID: course.ID})
		records, err := dao.ListCourseProgress(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, course.ID, records[0].CourseID)
	})

	// Test successfully retrieving paginated course progress records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 17 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			courses = append(courses, course)
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

			assetProgress := &models.AssetProgress{
				AssetID:  asset.ID,
				Position: 5,
			}
			require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().WithPagination(pagination.New(1, 10))
		records, err := dao.ListCourseProgress(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, courses[0].ID, records[0].CourseID)
		require.Equal(t, courses[9].ID, records[9].CourseID)

		// Second page with remaining 7 records
		p = NewOptions().WithPagination(pagination.New(2, 10))
		records, err = dao.ListCourseProgress(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, courses[10].ID, records[0].CourseID)
		require.Equal(t, courses[16].ID, records[6].CourseID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteCourseProgress(t *testing.T) {
	// Test successfully deleting a course progress record
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
			Position: 5,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_PROGRESS_TABLE_COURSE_ID: course.ID})
		require.Nil(t, dao.DeleteCourseProgress(ctx, opts))

		records, err := dao.ListCourseProgress(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent course progress record
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
			Position: 5,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_PROGRESS_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteCourseProgress(ctx, opts))

		records, err := dao.ListCourseProgress(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, course.ID, records[0].CourseID)
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
			Position: 5,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		require.ErrorIs(t, dao.DeleteCourseProgress(ctx, nil), utils.ErrWhere)

		records, err := dao.ListCourseProgress(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, course.ID, records[0].CourseID)
	})

	// Test cascading delete of course progress records when deleting a course
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
			Position: 5,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		require.Nil(t, dao.DeleteCourses(ctx, dbOpts))

		records, err := dao.ListCourseProgress(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)
	})
}
