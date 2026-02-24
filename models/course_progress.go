package models

import (
	"fmt"

	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	COURSE_PROGRESS_TABLE = "courses_progress"

	COURSE_PROGRESS_COURSE_ID    = "course_id"
	COURSE_PROGRESS_USER_ID      = "user_id"
	COURSE_PROGRESS_STARTED      = "started"
	COURSE_PROGRESS_STARTED_AT   = "started_at"
	COURSE_PROGRESS_PERCENT      = "percent"
	COURSE_PROGRESS_COMPLETED_AT = "completed_at"

	COURSE_PROGRESS_TABLE_ID           = COURSE_PROGRESS_TABLE + "." + BASE_ID
	COURSE_PROGRESS_TABLE_CREATED_AT   = COURSE_PROGRESS_TABLE + "." + BASE_CREATED_AT
	COURSE_PROGRESS_TABLE_UPDATED_AT   = COURSE_PROGRESS_TABLE + "." + BASE_UPDATED_AT
	COURSE_PROGRESS_TABLE_COURSE_ID    = COURSE_PROGRESS_TABLE + "." + COURSE_PROGRESS_COURSE_ID
	COURSE_PROGRESS_TABLE_USER_ID      = COURSE_PROGRESS_TABLE + "." + COURSE_PROGRESS_USER_ID
	COURSE_PROGRESS_TABLE_STARTED      = COURSE_PROGRESS_TABLE + "." + COURSE_PROGRESS_STARTED
	COURSE_PROGRESS_TABLE_STARTED_AT   = COURSE_PROGRESS_TABLE + "." + COURSE_PROGRESS_STARTED_AT
	COURSE_PROGRESS_TABLE_PERCENT      = COURSE_PROGRESS_TABLE + "." + COURSE_PROGRESS_PERCENT
	COURSE_PROGRESS_TABLE_COMPLETED_AT = COURSE_PROGRESS_TABLE + "." + COURSE_PROGRESS_COMPLETED_AT
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseProgress defines the model for a course progress
type CourseProgress struct {
	Base
	CourseID    string         `db:"course_id"`    // Immutable
	UserID      string         `db:"user_id"`      // Immutable
	Started     bool           `db:"started"`      // Mutable
	StartedAt   types.DateTime `db:"started_at"`   // Mutable
	Percent     int            `db:"percent"`      // Mutable
	CompletedAt types.DateTime `db:"completed_at"` // Mutable
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseProgressColumns returns the columns for use in a SELECT query
func CourseProgressColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_COURSE_ID, COURSE_PROGRESS_COURSE_ID),
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_USER_ID, COURSE_PROGRESS_USER_ID),
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_STARTED, COURSE_PROGRESS_STARTED),
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_STARTED_AT, COURSE_PROGRESS_STARTED_AT),
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_PERCENT, COURSE_PROGRESS_PERCENT),
		fmt.Sprintf("%s AS %s", COURSE_PROGRESS_TABLE_COMPLETED_AT, COURSE_PROGRESS_COMPLETED_AT),
	}
}

