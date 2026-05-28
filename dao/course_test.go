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

func Test_CreateCourse(t *testing.T) {
	// Test successfully inserting a course record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)
		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))
	})

	// Test successfully inserting a course record with a description
	t.Run("success with description", func(t *testing.T) {
		dao, ctx := setup(t)
		course := &models.Course{
			Title:       "Course 1",
			Path:        "/course-1",
			Description: "A test course description",
		}
		require.NoError(t, dao.CreateCourse(ctx, course))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		record, err := dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Equal(t, "A test course description", record.Description)
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)
		require.ErrorIs(t, dao.CreateCourse(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to duplicate record
	t.Run("duplicate", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Base: models.Base{ID: "1"}, Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		// Duplicate ID
		course = &models.Course{Base: models.Base{ID: "1"}, Title: "Course 2", Path: "/course-2"}
		require.ErrorContains(t, dao.CreateCourse(ctx, course), "UNIQUE constraint failed: "+models.COURSE_TABLE_ID)

		// Duplicate Path
		course = &models.Course{Base: models.Base{ID: "2"}, Title: "Course 2", Path: "/course-1"}
		require.ErrorContains(t, dao.CreateCourse(ctx, course), "UNIQUE constraint failed: "+models.COURSE_TABLE_PATH)
	})

	// Test error due to invalid fields
	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "", Path: ""}
		require.ErrorIs(t, dao.CreateCourse(ctx, course), utils.ErrTitle)

		course = &models.Course{Title: "Course 1", Path: ""}
		require.ErrorIs(t, dao.CreateCourse(ctx, course), utils.ErrPath)
	})

}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetCourse(t *testing.T) {
	// Test successfully retrieving a course record with no relations
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		record, err := dao.GetCourse(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, course.ID, record.ID)
		require.Nil(t, record.Progress)
	})

	// Test successfully retrieving a course record with relations
	t.Run("success with relations", func(t *testing.T) {
		dao, ctx := setup(t)

		// Create course
		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		// Read course (no progress yet) for default user (principal from setup)
		dbOpts := NewOptions().
			WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID}).
			WithUserProgress()

		record, err := dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Equal(t, course.ID, record.ID)
		require.Nil(t, record.Progress)

		// Create lesson + asset
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
			Weight:   1,
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		// Attach video metadata so fractional progress can be computed
		require.NoError(t, dao.CreateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID: asset.ID,
			VideoMetadata: &models.VideoMetadata{
				DurationSec: 60, // 60s total, so 30s -> 50%
				Container:   "mp4",
				MIMEType:    "video/mp4",
				VideoCodec:  "h264",
				Width:       1280,
				Height:      720,
				FPSNum:      30,
				FPSDen:      1,
			},
		}))

		// User 1: partial progress (position=30 of 60 => ~50%)
		assetProgress := &models.AssetProgress{
			AssetID:  asset.ID,
			Position: 30,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		// Read course progress for user 1 (should be ~50%)
		record, err = dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record.Progress)
		require.True(t, record.Progress.Started)
		require.Equal(t, 50, record.Progress.Percent)
		require.False(t, record.Progress.StartedAt.IsZero())
		require.True(t, record.Progress.CompletedAt.IsZero())

		// Create a second user
		user2 := &models.User{
			Username:     "user2",
			DisplayName:  "User 2",
			PasswordHash: "hash",
			Role:         types.UserRoleUser,
		}
		require.NoError(t, dao.CreateUser(ctx, user2))

		// Switch principal in ctx to user2
		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		principal.UserID = user2.ID
		ctx = context.WithValue(ctx, types.PrincipalContextKey, principal)

		// User 2: mark the same asset completed (-> 100%)
		assetProgress2 := &models.AssetProgress{
			AssetID:   asset.ID,
			Completed: true,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress2))

		// Confirm there are 2 asset_progress rows (one per user)
		builderOpts := newBuilderOptions(models.ASSET_PROGRESS_TABLE)
		count, err := countGeneric(ctx, dao, *builderOpts)
		require.NoError(t, err)
		require.Equal(t, 2, count)

		// Read course for user 2; should be 100%
		record, err = dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Equal(t, course.ID, record.ID)
		require.NotNil(t, record.Progress)
		require.True(t, record.Progress.Started)
		require.Equal(t, 100, record.Progress.Percent)
		require.False(t, record.Progress.StartedAt.IsZero())
		require.False(t, record.Progress.CompletedAt.IsZero())
	})

	// Test no error when retrieving a non-existent course record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetCourse(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})

	// Test error due to missing principal
	t.Run("missing principal", func(t *testing.T) {
		dao, _ := setup(t)

		dbOpts := NewOptions().WithUserProgress()
		record, err := dao.GetCourse(context.Background(), dbOpts)
		require.ErrorIs(t, err, utils.ErrPrincipal)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListCourses(t *testing.T) {
	// Test successfully retrieving all course records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			courses = append(courses, course)
			require.NoError(t, dao.CreateCourse(ctx, course))
			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListCourses(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courses[i].ID, record.ID)
			require.Nil(t, record.Progress)
		}
	})

	// Test successfully retrieving all course records with relations
	t.Run("success with relations", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			courses = append(courses, course)
			require.NoError(t, dao.CreateCourse(ctx, course))
			time.Sleep(1 * time.Millisecond)
		}

		dbOpts := NewOptions().
			WithUserProgress().
			WithOrderBy(models.COURSE_TABLE_CREATED_AT + " ASC")

		records, err := dao.ListCourses(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		// Ensure no progress when none exists
		for i, record := range records {
			require.Equal(t, courses[i].ID, record.ID)
			require.Nil(t, record.Progress)
		}

		// Generate progress for the default user
		assets := []*models.Asset{}
		for i, course := range courses {
			lesson := &models.Lesson{
				CourseID: course.ID,
				Title:    "Asset Group 1",
				Prefix:   sql.NullInt16{Int16: 1, Valid: true},
				Module:   "Module 1",
			}
			require.NoError(t, dao.CreateLesson(ctx, lesson))

			// Create Asset
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
			assets = append(assets, asset)
			require.NoError(t, dao.CreateAsset(ctx, asset))

			// Create an asset progress for the first course
			if i == 0 {
				assetProgress := &models.AssetProgress{
					AssetID:   asset.ID,
					Completed: true,
				}
				require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))
			}

			time.Sleep(1 * time.Millisecond)
		}

		// List again
		records, err = dao.ListCourses(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		// Ensure first course has progress, others do not
		for i, record := range records {
			require.Equal(t, courses[i].ID, record.ID)
			if i == 0 {
				require.NotNil(t, record.Progress)
				require.True(t, record.Progress.Started)
				require.Equal(t, 100, record.Progress.Percent)
				require.False(t, record.Progress.StartedAt.IsZero())
				require.False(t, record.Progress.CompletedAt.IsZero())
			} else {
				require.Nil(t, record.Progress)
			}
		}

		// Create another user
		user2 := &models.User{
			Username:     "user2",
			DisplayName:  "User 2",
			PasswordHash: "hash",
			Role:         types.UserRoleUser,
		}
		require.NoError(t, dao.CreateUser(ctx, user2))

		// Set the principal to user2, which is picked up when interacting with progress
		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		principal.UserID = user2.ID
		ctx = context.WithValue(ctx, types.PrincipalContextKey, principal)

		// For course 2, create an asset progress (and therefore another course progress) for the
		// new user
		assetProgress2 := &models.AssetProgress{
			AssetID:   assets[1].ID,
			Completed: true,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress2))

		// List again
		records, err = dao.ListCourses(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		// Ensure second course has progress for user2, others do not
		for i, record := range records {
			require.Equal(t, courses[i].ID, record.ID)
			if i == 1 {
				require.NotNil(t, record.Progress)
				require.True(t, record.Progress.Started)
				require.Equal(t, 100, record.Progress.Percent)
				require.False(t, record.Progress.StartedAt.IsZero())
				require.False(t, record.Progress.CompletedAt.IsZero())
			} else {
				require.Nil(t, record.Progress)
			}
		}
	})

	// Test no error when retrieving no course records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListCourses(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered course records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			courses = append(courses, course)
			require.NoError(t, dao.CreateCourse(ctx, course))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.COURSE_TABLE_CREATED_AT + " DESC")
		records, err := dao.ListCourses(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courses[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.COURSE_TABLE_CREATED_AT + " ASC")
		records, err = dao.ListCourses(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courses[i].ID, record.ID)
		}
	})

	// Test successfully retrieving selected course records
	t.Run("where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		records, err := dao.ListCourses(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, course.ID, records[0].ID)
	})

	// Test successfully retrieving paginated course records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 17 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			require.NoError(t, dao.CreateCourse(ctx, course))
			courses = append(courses, course)
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().
			WithOrderBy(models.COURSE_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(1, 10))

		records, err := dao.ListCourses(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, courses[0].ID, records[0].ID)
		require.Equal(t, courses[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().
			WithOrderBy(models.COURSE_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(2, 10))

		records, err = dao.ListCourses(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, courses[10].ID, records[0].ID)
		require.Equal(t, courses[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpdateCourse(t *testing.T) {
	// Test successfully updating a course record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		originalCourse := &models.Course{
			Title:       "Course 1",
			Path:        "/course-1",
			CardPath:    "/course-1/card-1",
			Available:   true,
			Duration:    100,
			InitialScan: true,
			Maintenance: false,
		}
		require.NoError(t, dao.CreateCourse(ctx, originalCourse))

		time.Sleep(1 * time.Millisecond)

		updatedCourse := &models.Course{
			Base:        originalCourse.Base,
			Title:       "Course 2",
			Path:        "/course-2",
			CardPath:    "/course-2/card-1",
			Available:   false,
			Duration:    200,
			InitialScan: false,
			Maintenance: true,
		}
		require.NoError(t, dao.UpdateCourse(ctx, updatedCourse))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: originalCourse.ID})
		record, err := dao.GetCourse(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, originalCourse.ID, record.ID)                    // No change
		require.True(t, record.CreatedAt.Equal(originalCourse.CreatedAt)) // No change
		require.Equal(t, updatedCourse.Title, record.Title)               // Changed
		require.Equal(t, updatedCourse.Path, record.Path)                 // Changed
		require.Equal(t, updatedCourse.CardPath, record.CardPath)         // Changed
		require.Equal(t, updatedCourse.Available, record.Available)       // Changed
		require.Equal(t, updatedCourse.Duration, record.Duration)         // Changed
		require.Equal(t, updatedCourse.InitialScan, record.InitialScan)   // Changed
		require.Equal(t, updatedCourse.Maintenance, record.Maintenance)   // Changed
		require.NotEqual(t, originalCourse.UpdatedAt, record.UpdatedAt)   // Changed
	})

	// Test successfully updating a course record with a description
	t.Run("success with description", func(t *testing.T) {
		dao, ctx := setup(t)

		originalCourse := &models.Course{
			Title:       "Course 1",
			Path:        "/course-1",
			Description: "Original description",
		}
		require.NoError(t, dao.CreateCourse(ctx, originalCourse))

		time.Sleep(1 * time.Millisecond)

		updatedCourse := &models.Course{
			Base:        originalCourse.Base,
			Title:       originalCourse.Title,
			Path:        originalCourse.Path,
			Description: "Updated description",
		}
		require.NoError(t, dao.UpdateCourse(ctx, updatedCourse))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: originalCourse.ID})
		record, err := dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Equal(t, "Updated description", record.Description)
	})

	// Test error due to invalid fields
	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		// Empty ID
		course.ID = ""
		require.ErrorIs(t, dao.UpdateCourse(ctx, course), utils.ErrId)

		// Invalid title
		course.ID = "1234"
		course.Title = ""
		require.ErrorIs(t, dao.UpdateCourse(ctx, course), utils.ErrTitle)

		// Invalid path
		course.Title = "Course 1"
		course.Path = ""
		require.ErrorIs(t, dao.UpdateCourse(ctx, course), utils.ErrPath)

		// Nil Model
		require.ErrorIs(t, dao.UpdateCourse(ctx, nil), utils.ErrNilPtr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteCourses(t *testing.T) {
	// Test successfully deleting a course record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		require.Nil(t, dao.DeleteCourses(ctx, opts))

		records, err := dao.ListCourses(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent course record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteCourses(ctx, opts))

		records, err := dao.ListCourses(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, course.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		require.ErrorIs(t, dao.DeleteCourses(ctx, nil), utils.ErrWhere)

		records, err := dao.ListCourses(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, course.ID, records[0].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ClassifyCoursePaths(t *testing.T) {
	// Test successfully classifying course paths
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 3 {
			c := &models.Course{
				Title: fmt.Sprintf("Course %d", i),
				Path:  fmt.Sprintf("/course-%d", i),
			}
			require.NoError(t, dao.CreateCourse(ctx, c))
			courses = append(courses, c)
		}

		path1 := "/"                       // ancestor
		path2 := "/test"                   // none
		path3 := courses[2].Path           // course
		path4 := courses[2].Path + "/test" // descendant

		result, err := dao.ClassifyCoursePaths(ctx, []string{path1, path2, path3, path4})
		require.Nil(t, err)

		require.Equal(t, types.PathClassificationAncestor, result[path1])
		require.Equal(t, types.PathClassificationNone, result[path2])
		require.Equal(t, types.PathClassificationCourse, result[path3])
		require.Equal(t, types.PathClassificationDescendant, result[path4])
	})

	// Test no error when classifying no paths
	t.Run("no paths", func(t *testing.T) {
		dao, ctx := setup(t)

		result, err := dao.ClassifyCoursePaths(ctx, []string{})
		require.Nil(t, err)
		require.Empty(t, result)
	})

	// Test no error when classifying empty paths
	t.Run("empty path", func(t *testing.T) {
		dao, ctx := setup(t)

		result, err := dao.ClassifyCoursePaths(ctx, []string{"", "", ""})
		require.Nil(t, err)
		require.Empty(t, result)
	})

	// Test error due to database error
	t.Run("db error", func(t *testing.T) {
		dao, ctx := setup(t)

		_, err := dao.db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TABLE)
		require.Nil(t, err)

		result, err := dao.ClassifyCoursePaths(ctx, []string{"/"})
		require.ErrorContains(t, err, "no such table: "+models.COURSE_TABLE)
		require.Empty(t, result)
	})
}
