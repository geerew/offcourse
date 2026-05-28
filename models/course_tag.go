package models

import "fmt"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	COURSE_TAG_TABLE = "courses_tags"

	COURSE_TAG_TAG_ID    = "tag_id"
	COURSE_TAG_COURSE_ID = "course_id"

	COURSE_TAG_TABLE_ID         = COURSE_TAG_TABLE + "." + BASE_ID
	COURSE_TAG_TABLE_CREATED_AT = COURSE_TAG_TABLE + "." + BASE_CREATED_AT
	COURSE_TAG_TABLE_UPDATED_AT = COURSE_TAG_TABLE + "." + BASE_UPDATED_AT
	COURSE_TAG_TABLE_TAG_ID     = COURSE_TAG_TABLE + "." + COURSE_TAG_TAG_ID
	COURSE_TAG_TABLE_COURSE_ID  = COURSE_TAG_TABLE + "." + COURSE_TAG_COURSE_ID
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseTag defines the model for a course tag
type CourseTag struct {
	Base
	TagID    string `db:"tag_id"`    // Immutable
	CourseID string `db:"course_id"` // Immutable

	// Joins
	Course string `db:"course_title"` // Alias for course title
	Tag    string `db:"tag_tag"`      // Alias for tag
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseTagColumns returns the columns for use in a SELECT query
func CourseTagColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", COURSE_TAG_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", COURSE_TAG_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", COURSE_TAG_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", COURSE_TAG_TABLE_TAG_ID, COURSE_TAG_TAG_ID),
		fmt.Sprintf("%s AS %s", COURSE_TAG_TABLE_COURSE_ID, COURSE_TAG_COURSE_ID),
		// Join columns
		fmt.Sprintf("%s AS course_title", COURSE_TABLE_TITLE),
		fmt.Sprintf("%s AS tag_tag", TAG_TABLE_TAG),
	}
}
