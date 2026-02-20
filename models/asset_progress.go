package models

import (
	"database/sql"
	"fmt"

	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	ASSET_PROGRESS_TABLE = "assets_progress"

	ASSET_PROGRESS_ASSET_ID      = "asset_id"
	ASSET_PROGRESS_USER_ID       = "user_id"
	ASSET_PROGRESS_POSITION      = "position"
	ASSET_PROGRESS_PROGRESS_FRAC = "progress_frac"
	ASSET_PROGRESS_COMPLETED     = "completed"
	ASSET_PROGRESS_COMPLETED_AT  = "completed_at"

	ASSET_PROGRESS_TABLE_ID            = ASSET_PROGRESS_TABLE + "." + BASE_ID
	ASSET_PROGRESS_TABLE_CREATED_AT    = ASSET_PROGRESS_TABLE + "." + BASE_CREATED_AT
	ASSET_PROGRESS_TABLE_UPDATED_AT    = ASSET_PROGRESS_TABLE + "." + BASE_UPDATED_AT
	ASSET_PROGRESS_TABLE_ASSET_ID      = ASSET_PROGRESS_TABLE + "." + ASSET_PROGRESS_ASSET_ID
	ASSET_PROGRESS_TABLE_USER_ID       = ASSET_PROGRESS_TABLE + "." + ASSET_PROGRESS_USER_ID
	ASSET_PROGRESS_TABLE_POSITION      = ASSET_PROGRESS_TABLE + "." + ASSET_PROGRESS_POSITION
	ASSET_PROGRESS_TABLE_PROGRESS_FRAC = ASSET_PROGRESS_TABLE + "." + ASSET_PROGRESS_PROGRESS_FRAC
	ASSET_PROGRESS_TABLE_COMPLETED     = ASSET_PROGRESS_TABLE + "." + ASSET_PROGRESS_COMPLETED
	ASSET_PROGRESS_TABLE_COMPLETED_AT  = ASSET_PROGRESS_TABLE + "." + ASSET_PROGRESS_COMPLETED_AT
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AssetProgress defines the model for an asset progress
type AssetProgress struct {
	Base
	AssetID      string         `db:"asset_id"`      // Immutable
	UserID       string         `db:"user_id"`       // Immutable
	Position     int            `db:"position"`      // Mutable
	ProgressFrac float64        `db:"progress_frac"` // Mutable
	Completed    bool           `db:"completed"`     // Mutable
	CompletedAt  types.DateTime `db:"completed_at"`  // Mutable
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AssetProgressColumns returns the columns for use in a SELECT query
func AssetProgressColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_ASSET_ID, ASSET_PROGRESS_ASSET_ID),
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_USER_ID, ASSET_PROGRESS_USER_ID),
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_POSITION, ASSET_PROGRESS_POSITION),
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_PROGRESS_FRAC, ASSET_PROGRESS_PROGRESS_FRAC),
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_COMPLETED, ASSET_PROGRESS_COMPLETED),
		fmt.Sprintf("%s AS %s", ASSET_PROGRESS_TABLE_COMPLETED_AT, ASSET_PROGRESS_COMPLETED_AT),
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AssetProgressRow is used to scan joined asset progress rows. The values will zero
// out if no progress exists
type AssetProgressRow struct {
	Position     sql.NullInt64   `db:"progress_position"`
	ProgressFrac sql.NullFloat64 `db:"progress_progress_frac"`
	Completed    sql.NullBool    `db:"progress_completed"`
	CompletedAt  types.DateTime  `db:"progress_completed_at"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ToDomain converts AssetProgressRow to AssetProgress
func (r AssetProgressRow) ToDomain() *AssetProgress {
	return &AssetProgress{
		Position:     int(r.Position.Int64),
		ProgressFrac: r.ProgressFrac.Float64,
		Completed:    r.Completed.Bool,
		CompletedAt:  r.CompletedAt,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AssetProgressRowColumns returns the list of columns to use when populating `AssetProgressRow`
func AssetProgressRowColumns() []string {
	return []string{
		fmt.Sprintf("%s AS progress_position", ASSET_PROGRESS_TABLE_POSITION),
		fmt.Sprintf("%s AS progress_progress_frac", ASSET_PROGRESS_TABLE_PROGRESS_FRAC),
		fmt.Sprintf("%s AS progress_completed", ASSET_PROGRESS_TABLE_COMPLETED),
		fmt.Sprintf("%s AS progress_completed_at", ASSET_PROGRESS_TABLE_COMPLETED_AT),
	}
}
