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

func helper_createLessons(t *testing.T, ctx context.Context, dao *DAO, numCourses int) ([]*models.Course, []*models.Lesson, []*models.Asset, []*models.Attachment) {
	t.Helper()

	allCourses := []*models.Course{}
	allLessons := []*models.Lesson{}
	allAssets := []*models.Asset{}
	allAttachments := []*models.Attachment{}

	for i := 0; i < numCourses; i++ {
		course := &models.Course{Title: fmt.Sprintf("Course %d", i+1), Path: fmt.Sprintf("/course %d", i+1)}
		require.NoError(t, dao.CreateCourse(ctx, course))
		allCourses = append(allCourses, course)

		// Create 3 lessons with 3 assets and 2 attachments each, reversed
		for _, lessonIndex := range []int{3, 2, 1} {
			lessonPrefix := fmt.Sprintf("%02d", lessonIndex)

			lesson := &models.Lesson{
				CourseID: course.ID,
				Title:    fmt.Sprintf("Asset Group %d", lessonIndex),
				Prefix:   sql.NullInt16{Int16: int16(lessonIndex), Valid: true},
				Module:   fmt.Sprintf("Module %d", lessonIndex),
			}
			require.NoError(t, dao.CreateLesson(ctx, lesson))
			allLessons = append(allLessons, lesson)
			time.Sleep(1 * time.Millisecond)

			// 3 assets, reversed sub-prefix: 3,2,1
			for _, assetIndex := range []int{3, 2, 1} {
				asset := &models.Asset{
					CourseID:  course.ID,
					LessonID:  lesson.ID,
					Title:     fmt.Sprintf("Asset %d", assetIndex),
					Prefix:    sql.NullInt16{Int16: int16(assetIndex), Valid: true},
					SubPrefix: sql.NullInt16{Int16: int16(assetIndex), Valid: true},
					Module:    fmt.Sprintf("Module %d", assetIndex),
					Type:      types.MustAsset("mp4"),
					Path:      fmt.Sprintf("%s/%s asset {%02d}.mp4", course.Path, lessonPrefix, assetIndex),
				}
				require.NoError(t, dao.CreateAsset(ctx, asset))
				allAssets = append(allAssets, asset)
				time.Sleep(1 * time.Millisecond)
			}

			// Create 2 attachments, reversed: 2,1
			for _, n := range []int{2, 1} {
				attachment := &models.Attachment{
					CourseID: course.ID,
					LessonID: lesson.ID,
					Title:    fmt.Sprintf("%s Attachment %d", lessonPrefix, n),
					Path:     fmt.Sprintf("%s/%s attachment %d.pdf", course.Path, lessonPrefix, n),
				}
				require.NoError(t, dao.CreateAttachment(ctx, attachment))
				allAttachments = append(allAttachments, attachment)
				time.Sleep(1 * time.Millisecond)
			}
		}

		time.Sleep(1 * time.Millisecond)

	}

	return allCourses, allLessons, allAssets, allAttachments
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_CreateLesson(t *testing.T) {
	// Test successfully creating a lesson record
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
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)
		require.ErrorIs(t, dao.CreateLesson(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to invalid fields
	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{}
		require.ErrorIs(t, dao.CreateLesson(ctx, lesson), utils.ErrCourseId)

		lesson.CourseID = course.ID
		require.ErrorIs(t, dao.CreateLesson(ctx, lesson), utils.ErrTitle)

		lesson.Title = "Asset Group 1"
		require.ErrorIs(t, dao.CreateLesson(ctx, lesson), utils.ErrPrefix)

		lesson.Prefix = sql.NullInt16{Int16: 1, Valid: true}
		require.NoError(t, dao.CreateLesson(ctx, lesson))
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetLesson(t *testing.T) {
	// Test successfully retrieving a lesson record with no relations
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		_, allGroups, allAssets, allAttachments := helper_createLessons(t, ctx, dao, 1)

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.LESSON_TABLE_ID: allGroups[0].ID})
		record, err := dao.GetLesson(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, allGroups[0].ID, record.ID)

		require.Len(t, record.Assets, 3)
		require.Equal(t, allAssets[2].ID, record.Assets[0].ID)
		require.Equal(t, allAssets[1].ID, record.Assets[1].ID)
		require.Equal(t, allAssets[0].ID, record.Assets[2].ID)

		require.Len(t, record.Attachments, 2)
		require.Equal(t, allAttachments[1].ID, record.Attachments[0].ID)
		require.Equal(t, allAttachments[0].ID, record.Attachments[1].ID)
	})

	// Test no error when retrieving a non-existent lesson record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetLesson(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})

	// Test error due to missing principal
	t.Run("missing principal", func(t *testing.T) {
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

		dbOpts := NewOptions().WithUserProgress()
		record, err := dao.GetLesson(context.Background(), dbOpts)
		require.ErrorIs(t, err, utils.ErrPrincipal)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListLessons(t *testing.T) {
	// Test successfully retrieving all lesson records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		helper_createLessons(t, ctx, dao, 3)

		records, err := dao.ListLessons(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 9)

		require.Len(t, records[0].Attachments, 2)
		require.Len(t, records[0].Assets, 3)
	})

	// Test successfully retrieving no lesson records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListLessons(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving selected lesson records
	t.Run("where", func(t *testing.T) {
		dao, ctx := setup(t)

		courses, lessons, _, _ := helper_createLessons(t, ctx, dao, 3)

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.LESSON_TABLE_COURSE_ID: courses[1].ID})
		records, err := dao.ListLessons(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		require.Equal(t, lessons[5].ID, records[0].ID)
		require.Equal(t, lessons[4].ID, records[1].ID)
		require.Equal(t, lessons[3].ID, records[2].ID)
	})

	// Test successfully retrieving paginated lesson records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lessons := []*models.Lesson{}
		for i := range 17 {
			lesson := &models.Lesson{
				CourseID: course.ID,
				Title:    fmt.Sprintf("Asset Group %d", i),
				Prefix:   sql.NullInt16{Int16: int16(i), Valid: true},
				Module:   fmt.Sprintf("Module %d", i),
			}
			require.NoError(t, dao.CreateLesson(ctx, lesson))
			lessons = append(lessons, lesson)
		}

		// First page with 10 records
		p := NewOptions().
			WithOrderBy(models.LESSON_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(1, 10))

		records, err := dao.ListLessons(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, lessons[0].ID, records[0].ID)
		require.Equal(t, lessons[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().
			WithOrderBy(models.LESSON_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(2, 10))

		records, err = dao.ListLessons(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, lessons[10].ID, records[0].ID)
		require.Equal(t, lessons[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpdateLesson(t *testing.T) {
	// Test successfully updating a lesson record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		originalLesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, originalLesson))

		time.Sleep(1 * time.Millisecond)

		updatedLesson := &models.Lesson{
			Base:     originalLesson.Base,
			CourseID: course.ID,
			Title:    "Asset Group 2",
			Prefix:   sql.NullInt16{Int16: 2, Valid: true},
			Module:   "Module 2",
		}
		require.NoError(t, dao.UpdateLesson(ctx, updatedLesson))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.LESSON_TABLE_ID: originalLesson.ID})
		record, err := dao.GetLesson(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, originalLesson.ID, record.ID)                    // No change
		require.Equal(t, originalLesson.CourseID, record.CourseID)        // No change
		require.True(t, record.CreatedAt.Equal(originalLesson.CreatedAt)) // No change
		require.Equal(t, updatedLesson.Title, record.Title)               // Changed
		require.Equal(t, updatedLesson.Prefix, record.Prefix)             // Changed
		require.Equal(t, updatedLesson.Module, record.Module)             // Changed
		require.NotEqual(t, originalLesson.UpdatedAt, record.UpdatedAt)   // Changed
	})

	// Test error due to invalid fields
	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{}

		// Course ID
		require.ErrorIs(t, dao.UpdateLesson(ctx, lesson), utils.ErrCourseId)
		lesson.CourseID = course.ID

		// Title
		require.ErrorIs(t, dao.UpdateLesson(ctx, lesson), utils.ErrTitle)
		lesson.Title = "Asset 1"

		// Prefix
		require.ErrorIs(t, dao.UpdateLesson(ctx, lesson), utils.ErrPrefix)
		lesson.Prefix = sql.NullInt16{Int16: 1, Valid: true}

		// ID
		require.ErrorIs(t, dao.UpdateLesson(ctx, lesson), utils.ErrId)

		// Nil Model
		require.ErrorIs(t, dao.UpdateLesson(ctx, nil), utils.ErrNilPtr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteLesson(t *testing.T) {
	// Test successfully deleting a lesson record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		opts := NewOptions().WithWhere(squirrel.Eq{models.LESSON_TABLE_ID: lesson.ID})
		require.Nil(t, dao.DeleteLessons(ctx, opts))

		records, err := dao.ListLessons(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent lesson record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		opts := NewOptions().WithWhere(squirrel.Eq{models.LESSON_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteLessons(ctx, opts))

		records, err := dao.ListLessons(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, lesson.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		require.ErrorIs(t, dao.DeleteLessons(ctx, nil), utils.ErrWhere)

		records, err := dao.ListLessons(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, lesson.ID, records[0].ID)
	})

	// Test cascading delete of lesson records when deleting a course
	t.Run("cascade", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		require.Nil(t, dao.DeleteCourses(ctx, dbOpts))

		records, err := dao.ListLessons(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)
	})
}
