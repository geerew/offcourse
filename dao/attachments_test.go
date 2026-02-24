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
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_CreateAttachments(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))
	})

	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)

		require.ErrorIs(t, dao.CreateAttachment(ctx, nil), utils.ErrNilPtr)
	})

	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		// Empty title
		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Path:     "/course-1/attachment-1",
		}
		require.ErrorIs(t, dao.CreateAttachment(ctx, attachment), utils.ErrTitle)

		// Empty path
		attachment = &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
		}
		require.ErrorIs(t, dao.CreateAttachment(ctx, attachment), utils.ErrPath)

		// Invalid lesson ID
		attachment = &models.Attachment{
			CourseID: course.ID,
			LessonID: "invalid",
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.ErrorContains(t, dao.CreateAttachment(ctx, attachment), "FOREIGN KEY constraint failed")

		// Invalid course ID
		attachment = &models.Attachment{
			CourseID: "invalid",
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.ErrorContains(t, dao.CreateAttachment(ctx, attachment), "FOREIGN KEY constraint failed")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetAttachment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ATTACHMENT_TABLE_ID: attachment.ID})
		record, err := dao.GetAttachment(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, attachment.ID, record.ID)
	})

	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetAttachment(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListAttachments(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachments := []*models.Attachment{}
		for i := range 3 {
			attachment := &models.Attachment{
				CourseID: course.ID,
				LessonID: lesson.ID,
				Title:    fmt.Sprintf("Attachment %d", i),
				Path:     fmt.Sprintf("/course-1/attachment-%d", i),
			}
			attachments = append(attachments, attachment)
			require.NoError(t, dao.CreateAttachment(ctx, attachment))
			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListAttachments(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, attachments[i].ID, record.ID)
		}
	})

	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListAttachments(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachments := []*models.Attachment{}
		for i := range 3 {
			attachment := &models.Attachment{
				CourseID: course.ID,
				LessonID: lesson.ID,
				Title:    fmt.Sprintf("Attachment %d", i),
				Path:     fmt.Sprintf("/course-1/attachment-%d", i),
			}
			attachments = append(attachments, attachment)
			require.NoError(t, dao.CreateAttachment(ctx, attachment))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.ATTACHMENT_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListAttachments(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, attachments[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.ATTACHMENT_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListAttachments(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, attachments[i].ID, record.ID)
		}
	})

	t.Run("where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ATTACHMENT_TABLE_ID: attachment.ID})
		records, err := dao.ListAttachments(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, attachment.ID, records[0].ID)
	})

	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachments := []*models.Attachment{}
		for i := range 17 {
			attachment := &models.Attachment{
				CourseID: course.ID,
				LessonID: lesson.ID,
				Title:    fmt.Sprintf("Attachment %d", i),
				Path:     fmt.Sprintf("/course-1/attachment-%d", i),
			}
			attachments = append(attachments, attachment)
			require.NoError(t, dao.CreateAttachment(ctx, attachment))
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().WithPagination(pagination.New(1, 10))
		records, err := dao.ListAttachments(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, attachments[0].ID, records[0].ID)
		require.Equal(t, attachments[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().WithPagination(pagination.New(2, 10))
		records, err = dao.ListAttachments(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, attachments[10].ID, records[0].ID)
		require.Equal(t, attachments[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpdateAttachment(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		originalAttachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, originalAttachment))

		time.Sleep(1 * time.Millisecond)

		updatedAttachment := &models.Attachment{
			Base:     originalAttachment.Base,
			LessonID: "1234",                         // Immutable
			Title:    "Updated Attachment",           // Mutable
			Path:     "/course-1/updated-attachment", // Mutable
		}
		require.NoError(t, dao.UpdateAttachment(ctx, updatedAttachment))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ATTACHMENT_TABLE_ID: originalAttachment.ID})
		record, err := dao.GetAttachment(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, originalAttachment.ID, record.ID)                     // No change
		require.Equal(t, originalAttachment.LessonID, record.LessonID)         // No change
		require.True(t, record.CreatedAt.Equal(originalAttachment.CreatedAt))  // No change
		require.Equal(t, updatedAttachment.Title, record.Title)                // Changed
		require.Equal(t, updatedAttachment.Path, record.Path)                  // Changed
		require.False(t, record.UpdatedAt.Equal(originalAttachment.UpdatedAt)) // Changed
	})

	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))

		// Empty path
		attachment.Path = ""
		require.ErrorIs(t, dao.UpdateAttachment(ctx, attachment), utils.ErrPath)

		// Empty title
		attachment.Title = ""
		require.ErrorIs(t, dao.UpdateAttachment(ctx, attachment), utils.ErrTitle)

		// Empty ID
		attachment.ID = ""
		require.ErrorIs(t, dao.UpdateAttachment(ctx, attachment), utils.ErrId)

		// Nil Model
		require.ErrorIs(t, dao.UpdateAttachment(ctx, nil), utils.ErrNilPtr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteAttachments(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ATTACHMENT_TABLE_ID: attachment.ID})
		require.Nil(t, dao.DeleteAttachments(ctx, opts))

		records, err := dao.ListAttachments(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ATTACHMENT_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteAttachments(ctx, opts))

		records, err := dao.ListAttachments(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, attachment.ID, records[0].ID)
	})

	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))

		require.ErrorIs(t, dao.DeleteAttachments(ctx, nil), utils.ErrWhere)

		records, err := dao.ListAttachments(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, attachment.ID, records[0].ID)
	})

	t.Run("cascade course", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		require.Nil(t, dao.DeleteCourses(ctx, opts))

		records, err := dao.ListAttachments(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)

	})

	t.Run("cascade lesson", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true, CardPath: "/course-1/card-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		attachment := &models.Attachment{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Attachment 1",
			Path:     "/course-1/attachment-1",
		}
		require.NoError(t, dao.CreateAttachment(ctx, attachment))

		// TODO change to deleteLesson when donee
		// require.Nil(t, Delete(ctx, dao, lesson, nil))

		// records, err := dao.ListAttachments(ctx, nil)
		// require.NoError(t, err)
		// require.Empty(t, records)
	})
}
