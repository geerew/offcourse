package models

import "fmt"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	COURSE_TABLE = "courses"

	COURSE_TITLE         = "title"
	COURSE_PATH          = "path"
	COURSE_CARD_PATH     = "card_path"
	COURSE_CARD_HASH     = "card_hash"
	COURSE_CARD_MOD_TIME = "card_mod_time"
	COURSE_AVAILABLE     = "available"
	COURSE_DURATION      = "duration"
	COURSE_INITIAL_SCAN  = "initial_scan"
	COURSE_MAINTENANCE   = "maintenance"
	COURSE_DESCRIPTION   = "description"

	COURSE_TABLE_ID            = COURSE_TABLE + "." + BASE_ID
	COURSE_TABLE_CREATED_AT    = COURSE_TABLE + "." + BASE_CREATED_AT
	COURSE_TABLE_UPDATED_AT    = COURSE_TABLE + "." + BASE_UPDATED_AT
	COURSE_TABLE_TITLE         = COURSE_TABLE + "." + COURSE_TITLE
	COURSE_TABLE_PATH          = COURSE_TABLE + "." + COURSE_PATH
	COURSE_TABLE_CARD_PATH     = COURSE_TABLE + "." + COURSE_CARD_PATH
	COURSE_TABLE_CARD_HASH     = COURSE_TABLE + "." + COURSE_CARD_HASH
	COURSE_TABLE_CARD_MOD_TIME = COURSE_TABLE + "." + COURSE_CARD_MOD_TIME
	COURSE_TABLE_AVAILABLE     = COURSE_TABLE + "." + COURSE_AVAILABLE
	COURSE_TABLE_DURATION      = COURSE_TABLE + "." + COURSE_DURATION
	COURSE_TABLE_INITIAL_SCAN  = COURSE_TABLE + "." + COURSE_INITIAL_SCAN
	COURSE_TABLE_MAINTENANCE   = COURSE_TABLE + "." + COURSE_MAINTENANCE
	COURSE_TABLE_DESCRIPTION   = COURSE_TABLE + "." + COURSE_DESCRIPTION
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Course defines the model for a course
type Course struct {
	Base
	Title       string `db:"title"`         // Mutable
	Path        string `db:"path"`          // Mutable
	CardPath    string `db:"card_path"`     // Mutable
	CardHash    string `db:"card_hash"`     // Mutable
	CardModTime string `db:"card_mod_time"` // Mutable
	Available   bool   `db:"available"`     // Mutable
	Duration    int    `db:"duration"`      // Mutable
	InitialScan bool   `db:"initial_scan"`  // Mutable
	Maintenance bool   `db:"maintenance"`   // Mutable
	Description string `db:"description"`   // Mutable

	// Relation
	Progress   *CourseProgress `db:"-"`
	Favourited bool            `db:"-"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseColumns returns the columns for use in a SELECT query
func CourseColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", COURSE_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_TITLE, COURSE_TITLE),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_PATH, COURSE_PATH),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_CARD_PATH, COURSE_CARD_PATH),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_CARD_HASH, COURSE_CARD_HASH),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_CARD_MOD_TIME, COURSE_CARD_MOD_TIME),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_AVAILABLE, COURSE_AVAILABLE),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_DURATION, COURSE_DURATION),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_INITIAL_SCAN, COURSE_INITIAL_SCAN),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_MAINTENANCE, COURSE_MAINTENANCE),
		fmt.Sprintf("%s AS %s", COURSE_TABLE_DESCRIPTION, COURSE_DESCRIPTION),
	}
}

