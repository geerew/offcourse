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

func Test_CreateCourseFavourite(t *testing.T) {
	// Test successfully inserting a course favourite record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.Nil(t, dao.CreateCourseFavourite(ctx, courseFavourite))
		require.NotEmpty(t, courseFavourite.ID)

		// Verify it was created
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_FAVOURITE_TABLE_ID: courseFavourite.ID})
		record, err := dao.GetCourseFavourite(ctx, dbOpts)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.Equal(t, courseFavourite.ID, record.ID)
		require.Equal(t, course.ID, record.CourseID)
		require.Equal(t, user.ID, record.UserID)
	})

	// Test error due to duplicate record
	t.Run("duplicate", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.Nil(t, dao.CreateCourseFavourite(ctx, courseFavourite))

		// Try to create duplicate
		courseFavourite2 := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.ErrorContains(t, dao.CreateCourseFavourite(ctx, courseFavourite2), "UNIQUE constraint failed")
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)

		require.ErrorIs(t, dao.CreateCourseFavourite(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to invalid course ID
	t.Run("invalid course ID", func(t *testing.T) {
		dao, ctx := setup(t)

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: "", UserID: user.ID}
		require.ErrorIs(t, dao.CreateCourseFavourite(ctx, courseFavourite), utils.ErrCourseId)
	})

	// Test error due to invalid user ID
	t.Run("invalid user ID", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: ""}
		require.ErrorIs(t, dao.CreateCourseFavourite(ctx, courseFavourite), utils.ErrUserId)
	})

	// Test error due to foreign key constraint on course
	t.Run("foreign key constraint - course", func(t *testing.T) {
		dao, ctx := setup(t)

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: "invalid", UserID: user.ID}
		require.ErrorContains(t, dao.CreateCourseFavourite(ctx, courseFavourite), "FOREIGN KEY constraint failed")
	})

	// Test error due to foreign key constraint on user
	t.Run("foreign key constraint - user", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: "invalid"}
		require.ErrorContains(t, dao.CreateCourseFavourite(ctx, courseFavourite), "FOREIGN KEY constraint failed")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetCourseFavourite(t *testing.T) {
	// Test successfully retrieving a course favourite record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_FAVOURITE_TABLE_ID: courseFavourite.ID})
		record, err := dao.GetCourseFavourite(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, courseFavourite.ID, record.ID)
		require.Equal(t, course.ID, record.CourseID)
		require.Equal(t, user.ID, record.UserID)
	})

	// Test no error when retrieving a non-existent course favourite record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetCourseFavourite(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListCourseFavourites(t *testing.T) {
	// Test successfully retrieving all course favourite records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i+1), Path: fmt.Sprintf("/course-%d", i+1)}
			require.NoError(t, dao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourites := []*models.CourseFavourite{}
		for _, course := range courses {
			courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
			courseFavourites = append(courseFavourites, courseFavourite)
			require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))
			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListCourseFavourites(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courseFavourites[i].ID, record.ID)
			require.Equal(t, courses[i].ID, record.CourseID)
			require.Equal(t, user.ID, record.UserID)
		}
	})

	// Test no error when retrieving no course favourite records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListCourseFavourites(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered course favourite records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i+1), Path: fmt.Sprintf("/course-%d", i+1)}
			require.NoError(t, dao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourites := []*models.CourseFavourite{}
		for _, course := range courses {
			courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
			courseFavourites = append(courseFavourites, courseFavourite)
			require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.COURSE_FAVOURITE_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListCourseFavourites(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courseFavourites[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.COURSE_FAVOURITE_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListCourseFavourites(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courseFavourites[i].ID, record.ID)
		}
	})

	// Test successfully retrieving selected course favourite records
	t.Run("where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_FAVOURITE_TABLE_ID: courseFavourite.ID})
		records, err := dao.ListCourseFavourites(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, courseFavourite.ID, records[0].ID)
	})

	// Test successfully retrieving paginated course favourite records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 17 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i+1), Path: fmt.Sprintf("/course-%d", i+1)}
			require.NoError(t, dao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourites := []*models.CourseFavourite{}
		for _, course := range courses {
			courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
			courseFavourites = append(courseFavourites, courseFavourite)
			require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().WithPagination(pagination.New(1, 10))
		records, err := dao.ListCourseFavourites(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, courseFavourites[0].ID, records[0].ID)
		require.Equal(t, courseFavourites[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().WithPagination(pagination.New(2, 10))
		records, err = dao.ListCourseFavourites(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, courseFavourites[10].ID, records[0].ID)
		require.Equal(t, courseFavourites[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteCourseFavourites(t *testing.T) {
	// Test successfully deleting a course favourite record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_FAVOURITE_TABLE_ID: courseFavourite.ID})
		require.Nil(t, dao.DeleteCourseFavourites(ctx, opts))

		records, err := dao.ListCourseFavourites(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent course favourite record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_FAVOURITE_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteCourseFavourites(ctx, opts))

		records, err := dao.ListCourseFavourites(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, courseFavourite.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))

		require.ErrorIs(t, dao.DeleteCourseFavourites(ctx, nil), utils.ErrWhere)

		records, err := dao.ListCourseFavourites(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, courseFavourite.ID, records[0].ID)
	})

	// Test successfully deleting course favourites when deleting a course
	t.Run("cascade - course", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		require.Nil(t, dao.DeleteCourses(ctx, opts))

		records, err := dao.ListCourseFavourites(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test successfully deleting course favourites when deleting a user
	t.Run("cascade - user", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		user := &models.User{Username: "user1", DisplayName: "User 1", PasswordHash: "hash", Role: "user"}
		require.NoError(t, dao.CreateUser(ctx, user))

		courseFavourite := &models.CourseFavourite{CourseID: course.ID, UserID: user.ID}
		require.NoError(t, dao.CreateCourseFavourite(ctx, courseFavourite))

		opts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: user.ID})
		require.Nil(t, dao.DeleteUsers(ctx, opts))

		records, err := dao.ListCourseFavourites(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)
	})
}
