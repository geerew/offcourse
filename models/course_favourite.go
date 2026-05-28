package models

import "fmt"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	COURSE_FAVOURITE_TABLE = "courses_favourites"

	COURSE_FAVOURITE_COURSE_ID = "course_id"
	COURSE_FAVOURITE_USER_ID   = "user_id"

	COURSE_FAVOURITE_TABLE_ID         = COURSE_FAVOURITE_TABLE + "." + BASE_ID
	COURSE_FAVOURITE_TABLE_CREATED_AT = COURSE_FAVOURITE_TABLE + "." + BASE_CREATED_AT
	COURSE_FAVOURITE_TABLE_UPDATED_AT = COURSE_FAVOURITE_TABLE + "." + BASE_UPDATED_AT
	COURSE_FAVOURITE_TABLE_COURSE_ID  = COURSE_FAVOURITE_TABLE + "." + COURSE_FAVOURITE_COURSE_ID
	COURSE_FAVOURITE_TABLE_USER_ID    = COURSE_FAVOURITE_TABLE + "." + COURSE_FAVOURITE_USER_ID
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseFavourite defines the model for a course favourite
type CourseFavourite struct {
	Base
	CourseID string `db:"course_id"` // Immutable
	UserID   string `db:"user_id"`   // Immutable
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseFavouriteColumns returns the columns for use in a SELECT query
func CourseFavouriteColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", COURSE_FAVOURITE_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", COURSE_FAVOURITE_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", COURSE_FAVOURITE_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", COURSE_FAVOURITE_TABLE_COURSE_ID, COURSE_FAVOURITE_COURSE_ID),
		fmt.Sprintf("%s AS %s", COURSE_FAVOURITE_TABLE_USER_ID, COURSE_FAVOURITE_USER_ID),
	}
}
