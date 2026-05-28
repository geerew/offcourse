package models

import (
	"fmt"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	TAG_TABLE = "tags"

	TAG_TAG          = "tag"
	TAG_COURSE_COUNT = "course_count"

	TAG_TABLE_ID         = TAG_TABLE + "." + BASE_ID
	TAG_TABLE_CREATED_AT = TAG_TABLE + "." + BASE_CREATED_AT
	TAG_TABLE_UPDATED_AT = TAG_TABLE + "." + BASE_UPDATED_AT
	TAG_TABLE_TAG        = TAG_TABLE + "." + TAG_TAG
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Tag defines the model for a tag
type Tag struct {
	Base
	Tag string `db:"tag"` // Mutable

	// Aggregate fields
	CourseCount int `db:"course_count"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// TagColumns returns the columns for use in a SELECT query
func TagColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", TAG_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", TAG_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", TAG_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", TAG_TABLE_TAG, TAG_TAG),

		// Aggregate fields
		fmt.Sprintf("COUNT(%s) as %s", COURSE_TAG_TABLE_COURSE_ID, TAG_COURSE_COUNT),
	}
}
