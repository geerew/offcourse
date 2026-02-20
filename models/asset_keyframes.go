package models

import (
	"encoding/json"
	"fmt"
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

	AssetID    string `db:"asset_id"`    // Immutable
	Keyframes  string `db:"keyframes"`   // Mutable
	IsComplete bool   `db:"is_complete"` // Mutable

	// Populated from KeyframesJSON
	KeyframesSlice []float64 `db:"-"`
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

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MarshalKeyframes converts the KeyframesSlice from a slice to a JSON string and
// stores it in KeyframesJSON
//
// When the keyframes slice is nil, it sets KeyframesJSON to "[]"
func (ak *AssetKeyframes) MarshalKeyframes() error {
	if ak.KeyframesSlice == nil {
		ak.Keyframes = "[]"
		return nil
	}

	data, err := json.Marshal(ak.KeyframesSlice)
	if err != nil {
		return fmt.Errorf("failed to marshal keyframes: %w", err)
	}

	ak.Keyframes = string(data)
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UnmarshalKeyframes converts the KeyframesJSON string to a slice and stores
// it in the KeyframesSlice slice
//
// When the KeyframesJSON is an empty string, it sets the KeyframesSlice slice to an empty
// slice
func (ak *AssetKeyframes) UnmarshalKeyframes() error {
	if ak.Keyframes == "" {
		ak.KeyframesSlice = []float64{}
		return nil
	}

	if err := json.Unmarshal([]byte(ak.Keyframes), &ak.KeyframesSlice); err != nil {
		return fmt.Errorf("failed to unmarshal keyframes: %w", err)
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ValidateKeyframes validates that the keyframes are in ascending order and non-negative
func (ak *AssetKeyframes) ValidateKeyframes() error {
	if len(ak.KeyframesSlice) == 0 {
		return nil
	}

	// Check for negative timestamps
	for i, timestamp := range ak.KeyframesSlice {
		if timestamp < 0 {
			return fmt.Errorf("keyframe at index %d has negative timestamp: %f", i, timestamp)
		}
	}

	// Check for ascending order
	for i := 1; i < len(ak.KeyframesSlice); i++ {
		if ak.KeyframesSlice[i] <= ak.KeyframesSlice[i-1] {
			return fmt.Errorf("keyframes not in ascending order: %f <= %f at indices %d, %d",
				ak.KeyframesSlice[i], ak.KeyframesSlice[i-1], i, i-1)
		}
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetSegmentCount returns the number of segments that would be generated from these keyframes
func (ak *AssetKeyframes) GetSegmentCount() int {
	return len(ak.KeyframesSlice)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetSegmentDuration returns the duration of a specific segment. This is calculated as the
// difference between the current keyframe and the next keyframe
//
// Returns 0 when the segment index is out of bounds or it is the last segment
func (ak *AssetKeyframes) GetSegmentDuration(segmentIndex int) float64 {
	if segmentIndex < 0 || segmentIndex >= len(ak.KeyframesSlice) || segmentIndex == len(ak.KeyframesSlice)-1 {
		return 0
	}

	return ak.KeyframesSlice[segmentIndex+1] - ak.KeyframesSlice[segmentIndex]
}
