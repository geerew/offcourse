package models

import "fmt"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	PARAM_TABLE = "params"
	PARAM_KEY   = "key"
	PARAM_VALUE = "value"

	PARAM_TABLE_ID         = PARAM_TABLE + "." + BASE_ID
	PARAM_TABLE_CREATED_AT = PARAM_TABLE + "." + BASE_CREATED_AT
	PARAM_TABLE_UPDATED_AT = PARAM_TABLE + "." + BASE_UPDATED_AT
	PARAM_TABLE_KEY        = PARAM_TABLE + "." + PARAM_KEY
	PARAM_TABLE_VALUE      = PARAM_TABLE + "." + PARAM_VALUE
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Param defines the model for a parameter
type Param struct {
	Base
	Key   string `db:"key"`   // Immutable
	Value string `db:"value"` // Mutable
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ParamColumns returns the columns for use in a SELECT query
func ParamColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", PARAM_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", PARAM_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", PARAM_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", PARAM_TABLE_KEY, PARAM_KEY),
		fmt.Sprintf("%s AS %s", PARAM_TABLE_VALUE, PARAM_VALUE),
	}
}
