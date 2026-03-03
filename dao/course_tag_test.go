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

func Test_CreateCourseTag(t *testing.T) {
	// Test successfully creating course tags
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 2 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			require.NoError(t, dao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		tag1 := &models.Tag{Tag: "Tag1"}
		require.NoError(t, dao.CreateTag(ctx, tag1))

		// Using ID (tag exists)
		courseTag := &models.CourseTag{TagID: tag1.ID, CourseID: courses[0].ID}
		require.Nil(t, dao.CreateCourseTag(ctx, courseTag))

		// Using Tag (tag exists)
		courseTagExisting := &models.CourseTag{CourseID: courses[1].ID, Tag: "Tag1"}
		require.Nil(t, dao.CreateCourseTag(ctx, courseTagExisting))

		// Create (tag does not exist)
		courseTagCreate := &models.CourseTag{CourseID: courses[0].ID, Tag: "Tag2"}
		require.Nil(t, dao.CreateCourseTag(ctx, courseTagCreate))

		// Case insensitive
		courseTagCaseInsensitive := &models.CourseTag{CourseID: courses[1].ID, Tag: "tag2"}
		require.Nil(t, dao.CreateCourseTag(ctx, courseTagCaseInsensitive))

		// Asset 4 course tags and 2 tags
		courseTags, err := dao.ListCourseTags(ctx, nil)
		require.NoError(t, err)
		require.Len(t, courseTags, 4)

		tags, err := dao.ListTags(ctx, nil)
		require.NoError(t, err)
		require.Len(t, tags, 2)
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)

		require.ErrorIs(t, dao.CreateCourseTag(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to invalid tag ID
	t.Run("invalid tag ID", func(t *testing.T) {
		dao, ctx := setup(t)

		require.ErrorIs(t, dao.CreateCourseTag(ctx, &models.CourseTag{CourseID: "1234"}), utils.ErrTag)
	})

	// Test error due to invalid course ID
	t.Run("invalid course ID", func(t *testing.T) {
		dao, ctx := setup(t)

		tag := &models.Tag{Tag: "Go"}
		require.NoError(t, dao.CreateTag(ctx, tag))

		courseTag := &models.CourseTag{TagID: tag.ID, CourseID: "invalid"}
		require.ErrorContains(t, dao.CreateCourseTag(ctx, courseTag), "FOREIGN KEY constraint failed")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetCourseTag(t *testing.T) {
	// Test successfully retrieving a course tag record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseTag := &models.CourseTag{CourseID: course.ID, Tag: "Tag 1"}
		require.NoError(t, dao.CreateCourseTag(ctx, courseTag))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_ID: courseTag.ID})
		record, err := dao.GetCourseTag(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, courseTag.ID, record.ID)
		require.Equal(t, courseTag.Tag, record.Tag)
	})

	// Test successfully retrieving no course tag records
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetCourseTag(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListCourseTags(t *testing.T) {
	// Test successfully retrieving all course tag records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseTags := []*models.CourseTag{}

		for i := range 3 {
			courseTag := &models.CourseTag{CourseID: course.ID, Tag: fmt.Sprintf("Tag %d", i)}
			courseTags = append(courseTags, courseTag)
			require.NoError(t, dao.CreateCourseTag(ctx, courseTag))
			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListCourseTags(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courseTags[i].ID, record.ID)
			require.Equal(t, courseTags[i].Tag, record.Tag)
		}
	})

	// Test successfully retrieving no course tag records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListCourseTags(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered course tag records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseTags := []*models.CourseTag{}
		for i := range 3 {
			courseTag := &models.CourseTag{CourseID: course.ID, Tag: fmt.Sprintf("Tag %d", i)}
			courseTags = append(courseTags, courseTag)
			require.NoError(t, dao.CreateCourseTag(ctx, courseTag))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.COURSE_TAG_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListCourseTags(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courseTags[2-i].ID, record.ID)
			require.Equal(t, courseTags[2-i].Tag, record.Tag)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.COURSE_TAG_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListCourseTags(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, courseTags[i].ID, record.ID)
			require.Equal(t, courseTags[i].Tag, record.Tag)
		}
	})

	// Test successfully retrieving selected course tag records
	t.Run("where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseTag := &models.CourseTag{CourseID: course.ID, Tag: "Tag 1"}
		require.NoError(t, dao.CreateCourseTag(ctx, courseTag))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_ID: courseTag.ID})
		records, err := dao.ListCourseTags(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, courseTag.ID, records[0].ID)
	})

	// Test successfully retrieving paginated course tag records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseTags := []*models.CourseTag{}
		for i := range 17 {
			courseTag := &models.CourseTag{CourseID: course.ID, Tag: fmt.Sprintf("Tag %d", i)}
			courseTags = append(courseTags, courseTag)
			require.NoError(t, dao.CreateCourseTag(ctx, courseTag))
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().WithPagination(pagination.New(1, 10))
		records, err := dao.ListCourseTags(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, courseTags[0].ID, records[0].ID)
		require.Equal(t, courseTags[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().WithPagination(pagination.New(2, 10))
		records, err = dao.ListCourseTags(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, courseTags[10].ID, records[0].ID)
		require.Equal(t, courseTags[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteCourseTags(t *testing.T) {
	// Test successfully deleting a course tag record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseTag := &models.CourseTag{CourseID: course.ID, Tag: "Tag 1"}
		require.NoError(t, dao.CreateCourseTag(ctx, courseTag))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_ID: courseTag.ID})
		require.Nil(t, dao.DeleteCourseTags(ctx, opts))

		records, err := dao.ListCourseTags(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test successfully deleting a non-existent course tag record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseTag := &models.CourseTag{CourseID: course.ID, Tag: "Tag 1"}
		require.NoError(t, dao.CreateCourseTag(ctx, courseTag))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteCourseTags(ctx, opts))

		records, err := dao.ListCourseTags(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, courseTag.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		courseTag := &models.CourseTag{CourseID: course.ID, Tag: "Tag 1"}
		require.NoError(t, dao.CreateCourseTag(ctx, courseTag))

		require.ErrorIs(t, dao.DeleteCourseTags(ctx, nil), utils.ErrWhere)

		records, err := dao.ListCourseTags(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, courseTag.ID, records[0].ID)
	})

	// Test cascading delete of course tag records when deleting a course
	t.Run("cascade", func(t *testing.T) {
		dao, ctx := setup(t)

		// Delete course
		course1 := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course1))

		course1Tag := &models.CourseTag{CourseID: course1.ID, Tag: "Tag 1"}
		require.NoError(t, dao.CreateCourseTag(ctx, course1Tag))

		opts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course1.ID})
		require.Nil(t, dao.DeleteCourses(ctx, opts))

		records, err := dao.ListCourseTags(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)

		// Delete tag
		course2 := &models.Course{Title: "Course 2", Path: "/course-2"}
		require.NoError(t, dao.CreateCourse(ctx, course2))

		course2Tag := &models.CourseTag{CourseID: course2.ID, Tag: "Tag 2"}
		require.NoError(t, dao.CreateCourseTag(ctx, course2Tag))

		opts = NewOptions().WithWhere(squirrel.Eq{models.TAG_TABLE_ID: course2Tag.TagID})
		require.Nil(t, dao.DeleteTags(ctx, opts))

		records, err = dao.ListCourseTags(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)
	})
}
