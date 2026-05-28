package models

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

import (
	"database/sql"
	"fmt"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	LESSON_TABLE = "lessons"

	LESSON_COURSE_ID = "course_id"
	LESSON_TITLE     = "title"
	LESSON_PREFIX    = "prefix"
	LESSON_MODULE    = "module"

	LESSON_TABLE_ID         = LESSON_TABLE + "." + BASE_ID
	LESSON_TABLE_CREATED_AT = LESSON_TABLE + "." + BASE_CREATED_AT
	LESSON_TABLE_UPDATED_AT = LESSON_TABLE + "." + BASE_UPDATED_AT
	LESSON_TABLE_COURSE_ID  = LESSON_TABLE + "." + LESSON_COURSE_ID
	LESSON_TABLE_TITLE      = LESSON_TABLE + "." + LESSON_TITLE
	LESSON_TABLE_PREFIX     = LESSON_TABLE + "." + LESSON_PREFIX
	LESSON_TABLE_MODULE     = LESSON_TABLE + "." + LESSON_MODULE
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Lesson defines the model for a lesson
type Lesson struct {
	Base
	CourseID string        `db:"course_id"` // Immutable
	Title    string        `db:"title"`     // Mutable
	Prefix   sql.NullInt16 `db:"prefix"`    // Mutable
	Module   string        `db:"module"`    // Mutable

	// Relations
	Assets      []*Asset
	Attachments []*Attachment
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// LessonColumns returns the columns for use in a SELECT query
func LessonColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", LESSON_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", LESSON_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", LESSON_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", LESSON_TABLE_COURSE_ID, LESSON_COURSE_ID),
		fmt.Sprintf("%s AS %s", LESSON_TABLE_TITLE, LESSON_TITLE),
		fmt.Sprintf("%s AS %s", LESSON_TABLE_PREFIX, LESSON_PREFIX),
		fmt.Sprintf("%s AS %s", LESSON_TABLE_MODULE, LESSON_MODULE),
	}
}
