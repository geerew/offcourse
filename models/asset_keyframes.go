package models

import (
	"fmt"

	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	ASSET_KEYFRAMES_TABLE = "asset_keyframes"

	ASSET_KEYFRAMES_ASSET_ID    = "asset_id"
	ASSET_KEYFRAMES_KEYFRAMES   = "keyframes"
	ASSET_KEYFRAMES_IS_COMPLETE = "is_complete"

	ASSET_KEYFRAMES_TABLE_ID          = ASSET_KEYFRAMES_TABLE + "." + BASE_ID
	ASSET_KEYFRAMES_TABLE_CREATED_AT  = ASSET_KEYFRAMES_TABLE + "." + BASE_CREATED_AT
	ASSET_KEYFRAMES_TABLE_UPDATED_AT  = ASSET_KEYFRAMES_TABLE + "." + BASE_UPDATED_AT
	ASSET_KEYFRAMES_TABLE_ASSET_ID    = ASSET_KEYFRAMES_TABLE + "." + ASSET_KEYFRAMES_ASSET_ID
	ASSET_KEYFRAMES_TABLE_KEYFRAMES   = ASSET_KEYFRAMES_TABLE + "." + ASSET_KEYFRAMES_KEYFRAMES
	ASSET_KEYFRAMES_TABLE_IS_COMPLETE = ASSET_KEYFRAMES_TABLE + "." + ASSET_KEYFRAMES_IS_COMPLETE
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AssetKeyframes defines keyframe data for a video asset
type AssetKeyframes struct {
	Base

	AssetID    string          `db:"asset_id"`    // Immutable
	Keyframes  types.Keyframes `db:"keyframes"`   // Mutable
	IsComplete bool            `db:"is_complete"` // Mutable
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AssetKeyframesColumns returns the columns for use in a SELECT query
func AssetKeyframesColumns() []string {
	return []string{
		fmt.Sprintf("%s AS %s", ASSET_KEYFRAMES_TABLE_ID, BASE_ID),
		fmt.Sprintf("%s AS %s", ASSET_KEYFRAMES_TABLE_CREATED_AT, BASE_CREATED_AT),
		fmt.Sprintf("%s AS %s", ASSET_KEYFRAMES_TABLE_UPDATED_AT, BASE_UPDATED_AT),
		fmt.Sprintf("%s AS %s", ASSET_KEYFRAMES_TABLE_ASSET_ID, ASSET_KEYFRAMES_ASSET_ID),
		fmt.Sprintf("%s AS %s", ASSET_KEYFRAMES_TABLE_KEYFRAMES, ASSET_KEYFRAMES_KEYFRAMES),
		fmt.Sprintf("%s AS %s", ASSET_KEYFRAMES_TABLE_IS_COMPLETE, ASSET_KEYFRAMES_IS_COMPLETE),
	}
}
