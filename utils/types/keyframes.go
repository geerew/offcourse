package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Keyframes defines a slice of float64 timestamps
type Keyframes []float64

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Value implements the `driver.Valuer` interface
func (k Keyframes) Value() (driver.Value, error) {
	if k == nil {
		return "[]", nil
	}
	data, err := json.Marshal(k)
	return string(data), err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Scan implements the `sql.Scanner` interface
func (k *Keyframes) Scan(value any) error {
	var data []byte
	switch v := value.(type) {
	case nil:
		*k = []float64{}
		return nil
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("keyframes: unexpected type %T", value)
	}

	if len(data) == 0 {
		*k = []float64{}
		return nil
	}

	return json.Unmarshal(data, k)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Validate checks that keyframes are non-negative and in ascending order
func (k Keyframes) Validate() error {
	if len(k) == 0 {
		return nil
	}

	for i, timestamp := range k {
		if timestamp < 0 {
			return fmt.Errorf("keyframe at index %d has negative timestamp: %f", i, timestamp)
		}
	}

	for i := 1; i < len(k); i++ {
		if k[i] <= k[i-1] {
			return fmt.Errorf("keyframes not in ascending order: %f <= %f at indices %d, %d",
				k[i], k[i-1], i, i-1)
		}
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SegmentCount returns the number of segments
func (k Keyframes) SegmentCount() int {
	return len(k)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SegmentDuration returns the duration of the segment at index
//
// Returns 0 when the index is out of bounds or it's the last segment
func (k Keyframes) SegmentDuration(segmentIndex int) float64 {
	if segmentIndex < 0 || segmentIndex >= len(k) || segmentIndex == len(k)-1 {
		return 0
	}

	return k[segmentIndex+1] - k[segmentIndex]
}
