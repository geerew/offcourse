package models

import (
	"fmt"

	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
const (
	LOG_TABLE = "logs"

	LOG_LEVEL   = "level"
	LOG_MESSAGE = "message"
	LOG_DATA    = "data"

	LOG_TABLE_ID         = LOG_TABLE + "." + BASE_ID
	LOG_TABLE_CREATED_AT = LOG_TABLE + "." + BASE_CREATED_AT
	LOG_TABLE_UPDATED_AT = LOG_TABLE + "." + BASE_UPDATED_AT
	LOG_TABLE_LEVEL      = LOG_TABLE + "." + LOG_LEVEL
	LOG_TABLE_MESSAGE    = LOG_TABLE + "." + LOG_MESSAGE
	LOG_TABLE_DATA       = LOG_TABLE + "." + LOG_DATA
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Log defines the model for a log
type Log struct {
	Base
	Level   string        `db:"level"`   // Immutable
	Message string        `db:"message"` // Immutable
	Data    types.JsonMap `db:"data"`    // Immutable
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// LogColumns returns the columns for use in a SELECT query
func LogColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", LOG_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", LOG_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", LOG_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", LOG_TABLE_LEVEL, LOG_LEVEL),
		fmt.Sprintf("%s AS %s", LOG_TABLE_MESSAGE, LOG_MESSAGE),
		fmt.Sprintf("%s AS %s", LOG_TABLE_DATA, LOG_DATA),
	}
}
