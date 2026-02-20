package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAssetKeyframes_MarshalKeyframes(t *testing.T) {
	// Test successfully marshalling empty keyframes
	t.Run("empty keyframes", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{},
		}

		err := ak.MarshalKeyframes()
		require.NoError(t, err)
		assert.Equal(t, "[]", ak.Keyframes)
	})

	// Test successfully marshalling nil keyframes
	t.Run("nil keyframes", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: nil,
		}

		err := ak.MarshalKeyframes()
		require.NoError(t, err)
		assert.Equal(t, "[]", ak.Keyframes)
	})

	// Test successfully marshalling multiple keyframes
	t.Run("multiple keyframes", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{0.0, 2.5, 5.0, 7.5, 10.0},
		}

		err := ak.MarshalKeyframes()
		require.NoError(t, err)
		assert.Equal(t, "[0,2.5,5,7.5,10]", ak.Keyframes)
	})

	// Test successfully marshalling single keyframe
	t.Run("single keyframe", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{0.0},
		}

		err := ak.MarshalKeyframes()
		require.NoError(t, err)
		assert.Equal(t, "[0]", ak.Keyframes)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAssetKeyframes_UnmarshalKeyframes(t *testing.T) {
	// Test successfully unmarshalling empty JSON
	t.Run("empty JSON", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:   "test-asset",
			Keyframes: "",
		}

		err := ak.UnmarshalKeyframes()
		require.NoError(t, err)
		assert.Equal(t, []float64{}, ak.KeyframesSlice)
	})

	// Test successfully unmarshalling empty array JSON
	t.Run("empty array JSON", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:   "test-asset",
			Keyframes: "[]",
		}

		err := ak.UnmarshalKeyframes()
		require.NoError(t, err)
		assert.Equal(t, []float64{}, ak.KeyframesSlice)
	})

	// Test successfully unmarshalling valid JSON with multiple keyframes
	t.Run("valid JSON", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:   "test-asset",
			Keyframes: "[0,2.5,5,7.5,10]",
		}

		err := ak.UnmarshalKeyframes()
		require.NoError(t, err)
		assert.Equal(t, []float64{0.0, 2.5, 5.0, 7.5, 10.0}, ak.KeyframesSlice)
	})

	// Test error due to invalid JSON
	t.Run("invalid JSON", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:   "test-asset",
			Keyframes: "invalid json",
		}

		err := ak.UnmarshalKeyframes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal keyframes")
	})

	// Test error due to wrong type JSON
	t.Run("wrong type JSON", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:   "test-asset",
			Keyframes: `{"not": "array"}`,
		}

		err := ak.UnmarshalKeyframes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal keyframes")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAssetKeyframes_ValidateKeyframes(t *testing.T) {
	// Test successfully validating empty keyframes
	t.Run("empty keyframes", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{},
		}

		err := ak.ValidateKeyframes()
		require.NoError(t, err)
	})

	// Test successfully validating single keyframe
	t.Run("single keyframe", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{0.0},
		}

		err := ak.ValidateKeyframes()
		require.NoError(t, err)
	})

	// Test successfully validating valid ascending keyframes
	t.Run("valid ascending keyframes", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{0.0, 2.5, 5.0, 7.5, 10.0},
		}

		err := ak.ValidateKeyframes()
		require.NoError(t, err)
	})

	// Test error due to negative timestamp in keyframes
	t.Run("negative timestamp", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{-1.0, 2.5, 5.0},
		}

		err := ak.ValidateKeyframes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "negative timestamp")
	})

	// Test error due to not ascending order in keyframes
	t.Run("not ascending order", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{0.0, 5.0, 2.5, 10.0},
		}

		err := ak.ValidateKeyframes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in ascending order")
	})

	// Test error due to duplicate timestamps in keyframes
	t.Run("duplicate timestamps", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{0.0, 2.5, 2.5, 5.0},
		}

		err := ak.ValidateKeyframes()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in ascending order")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAssetKeyframes_GetSegmentCount(t *testing.T) {
	// Test successfully getting segment count for empty keyframes
	t.Run("empty keyframes", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{},
		}

		count := ak.GetSegmentCount()
		assert.Equal(t, 0, count)
	})

	// Test successfully getting segment count for single keyframe
	t.Run("single keyframe", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{0.0},
		}

		count := ak.GetSegmentCount()
		assert.Equal(t, 1, count)
	})

	// Test successfully getting segment count for multiple keyframes
	t.Run("multiple keyframes", func(t *testing.T) {
		ak := &AssetKeyframes{
			AssetID:        "test-asset",
			KeyframesSlice: []float64{0.0, 2.5, 5.0, 7.5, 10.0},
		}

		count := ak.GetSegmentCount()
		assert.Equal(t, 5, count)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestAssetKeyframes_GetSegmentDuration(t *testing.T) {
	ak := &AssetKeyframes{
		AssetID:        "test-asset",
		KeyframesSlice: []float64{0.0, 2.5, 5.0, 7.5, 10.0},
	}

	// Test returning 0 for negative index
	t.Run("negative index", func(t *testing.T) {
		assert.Equal(t, 0.0, ak.GetSegmentDuration(-1))
	})

	// Test returning 0 for index out of bounds
	t.Run("index out of bounds", func(t *testing.T) {
		assert.Equal(t, 0.0, ak.GetSegmentDuration(10))
	})

	// Test successfully getting valid segment duration
	t.Run("valid segments", func(t *testing.T) {
		assert.Equal(t, 2.5, ak.GetSegmentDuration(0))
		assert.Equal(t, 2.5, ak.GetSegmentDuration(1))
		assert.Equal(t, 2.5, ak.GetSegmentDuration(2))
		assert.Equal(t, 2.5, ak.GetSegmentDuration(3))
	})

	// Test returning 0 for last segment (when there is no total video duration)
	t.Run("last segment", func(t *testing.T) {
		assert.Equal(t, 0.0, ak.GetSegmentDuration(4))
	})
}
