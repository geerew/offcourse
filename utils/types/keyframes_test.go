package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the keyframes value
func TestKeyframes_Value(t *testing.T) {
	scenarios := []struct {
		keyframes Keyframes
		expected  string
	}{
		{nil, "[]"},
		{Keyframes{}, "[]"},
		{Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}, "[0,2.5,5,7.5,10]"},
		{Keyframes{0.0}, "[0]"},
	}

	for i, s := range scenarios {
		v, err := s.keyframes.Value()
		require.NoError(t, err)
		require.Equal(t, s.expected, v, "(%d) Expected %s, got %s", i, s.expected, v)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_Scan(t *testing.T) {
	// Test successfully scanning keyframes from various inputs
	t.Run("success", func(t *testing.T) {
		scenarios := []struct {
			value    any
			expected Keyframes
		}{
			{nil, Keyframes{}},
			{"", Keyframes{}},
			{[]byte{}, Keyframes{}},
			{`[]`, Keyframes{}},
			{`[0,2.5,5,7.5,10]`, Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}},
			{`[0]`, Keyframes{0.0}},
		}

		for i, s := range scenarios {
			var k Keyframes
			err := k.Scan(s.value)
			require.NoError(t, err, "(%d) Expected no error, got %v", i, err)
			require.Equal(t, s.expected, k, "(%d) Expected %v, got %v", i, s.expected, k)
		}
	})

	// Test erroring when an invalid JSON is provided
	t.Run("error", func(t *testing.T) {
		scenarios := []any{
			`invalid json`,
			`{"not": "array"}`,
		}

		for i, s := range scenarios {
			var k Keyframes
			err := k.Scan(s)
			require.Error(t, err, "(%d) Expected error, got %v", i, err)
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_Validate(t *testing.T) {
	// Test successfully validating keyframes
	t.Run("success", func(t *testing.T) {
		scenarios := []Keyframes{
			{},
			{0.0},
			{0.0, 2.5, 5.0, 7.5, 10.0},
		}

		for i, k := range scenarios {
			err := k.Validate()
			require.NoError(t, err, "(%d) Expected no error, got %v", i, err)
		}
	})

	// Test erroring when keyframes are not in ascending order, or contain negative
	// timestamps, or duplicate timestamps
	t.Run("error", func(t *testing.T) {
		scenarios := []Keyframes{
			{-1.0, 2.5, 5.0},
			{0.0, 5.0, 2.5, 10.0},
			{0.0, 2.5, 2.5, 5.0},
		}

		for i, k := range scenarios {
			err := k.Validate()
			require.Error(t, err, "(%d) Expected error, got %v", i, err)
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully getting the number of segments
func TestKeyframes_SegmentCount(t *testing.T) {
	scenarios := []struct {
		keyframes Keyframes
		expected  int
	}{
		{Keyframes{}, 0},
		{Keyframes{0.0}, 1},
		{Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}, 5},
	}

	for i, k := range scenarios {
		assert.Equal(t, k.expected, k.keyframes.SegmentCount(), "(%d) Expected %d, got %d", i, k.expected, k.keyframes.SegmentCount())
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_SegmentDuration(t *testing.T) {
	k := Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}

	// Test successfully getting 0.0 when the index is negative
	t.Run("negative index", func(t *testing.T) {
		assert.Equal(t, 0.0, k.SegmentDuration(-1))
	})

	// Test successfully getting 0.0 when the index is out of bounds
	t.Run("index out of bounds", func(t *testing.T) {
		assert.Equal(t, 0.0, k.SegmentDuration(10))
	})

	// Test successfully getting the duration of the segment at the given index
	t.Run("valid segments", func(t *testing.T) {
		assert.Equal(t, 2.5, k.SegmentDuration(0))
		assert.Equal(t, 2.5, k.SegmentDuration(1))
		assert.Equal(t, 2.5, k.SegmentDuration(2))
		assert.Equal(t, 2.5, k.SegmentDuration(3))
	})

	// Test successfully getting 0.0 when the index is the last segment
	t.Run("last segment", func(t *testing.T) {
		assert.Equal(t, 0.0, k.SegmentDuration(4))
	})
}
