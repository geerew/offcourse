package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_Value(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var k Keyframes
		v, err := k.Value()
		require.NoError(t, err)
		assert.Equal(t, "[]", v)
	})

	t.Run("empty", func(t *testing.T) {
		k := Keyframes{}
		v, err := k.Value()
		require.NoError(t, err)
		assert.Equal(t, "[]", v)
	})

	t.Run("multiple", func(t *testing.T) {
		k := Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}
		v, err := k.Value()
		require.NoError(t, err)
		assert.Equal(t, "[0,2.5,5,7.5,10]", v)
	})

	t.Run("single", func(t *testing.T) {
		k := Keyframes{0.0}
		v, err := k.Value()
		require.NoError(t, err)
		assert.Equal(t, "[0]", v)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_Scan(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var k Keyframes
		err := k.Scan(nil)
		require.NoError(t, err)
		assert.Equal(t, Keyframes{}, k)
	})

	t.Run("empty string", func(t *testing.T) {
		var k Keyframes
		err := k.Scan("")
		require.NoError(t, err)
		assert.Equal(t, Keyframes{}, k)
	})

	t.Run("empty array JSON", func(t *testing.T) {
		var k Keyframes
		err := k.Scan("[]")
		require.NoError(t, err)
		assert.Equal(t, Keyframes{}, k)
	})

	t.Run("valid JSON", func(t *testing.T) {
		var k Keyframes
		err := k.Scan("[0,2.5,5,7.5,10]")
		require.NoError(t, err)
		assert.Equal(t, Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}, k)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		var k Keyframes
		err := k.Scan("invalid json")
		require.Error(t, err)
	})

	t.Run("wrong type JSON", func(t *testing.T) {
		var k Keyframes
		err := k.Scan(`{"not": "array"}`)
		require.Error(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_Validate(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		k := Keyframes{}
		require.NoError(t, k.Validate())
	})

	t.Run("single", func(t *testing.T) {
		k := Keyframes{0.0}
		require.NoError(t, k.Validate())
	})

	t.Run("valid ascending", func(t *testing.T) {
		k := Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}
		require.NoError(t, k.Validate())
	})

	t.Run("negative timestamp", func(t *testing.T) {
		k := Keyframes{-1.0, 2.5, 5.0}
		err := k.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "negative timestamp")
	})

	t.Run("not ascending", func(t *testing.T) {
		k := Keyframes{0.0, 5.0, 2.5, 10.0}
		err := k.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in ascending order")
	})

	t.Run("duplicate timestamps", func(t *testing.T) {
		k := Keyframes{0.0, 2.5, 2.5, 5.0}
		err := k.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in ascending order")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_SegmentCount(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		k := Keyframes{}
		assert.Equal(t, 0, k.SegmentCount())
	})

	t.Run("single", func(t *testing.T) {
		k := Keyframes{0.0}
		assert.Equal(t, 1, k.SegmentCount())
	})

	t.Run("multiple", func(t *testing.T) {
		k := Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}
		assert.Equal(t, 5, k.SegmentCount())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_SegmentDuration(t *testing.T) {
	k := Keyframes{0.0, 2.5, 5.0, 7.5, 10.0}

	t.Run("negative index", func(t *testing.T) {
		assert.Equal(t, 0.0, k.SegmentDuration(-1))
	})

	t.Run("index out of bounds", func(t *testing.T) {
		assert.Equal(t, 0.0, k.SegmentDuration(10))
	})

	t.Run("valid segments", func(t *testing.T) {
		assert.Equal(t, 2.5, k.SegmentDuration(0))
		assert.Equal(t, 2.5, k.SegmentDuration(1))
		assert.Equal(t, 2.5, k.SegmentDuration(2))
		assert.Equal(t, 2.5, k.SegmentDuration(3))
	})

	t.Run("last segment", func(t *testing.T) {
		assert.Equal(t, 0.0, k.SegmentDuration(4))
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestKeyframes_SqlRoundTrip(t *testing.T) {
	// Verify Value/Scan round-trip works with sql.Null
	original := Keyframes{0.0, 2.5, 5.0}
	v, err := original.Value()
	require.NoError(t, err)

	var scanned Keyframes
	err = scanned.Scan(v)
	require.NoError(t, err)
	assert.Equal(t, original, scanned)
}
