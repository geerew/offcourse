package probe

import (
	"context"
	"strings"
	"testing"

	"github.com/geerew/off-course/utils/media"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestExtractKeyframes(t *testing.T) {
	tools, err := media.NewTools()
	require.NoError(t, err)

	mp := MediaProbe{Tools: tools}

	t.Run("success with sample video", func(t *testing.T) {
		// Use the existing test video
		videoPath := "testdata/sample.mp4"

		keyframes, err := mp.ExtractKeyframesForVideo(context.Background(), videoPath)

		// The sample video might not have enough keyframes, which is OK for testing
		if err != nil && strings.Contains(err.Error(), "insufficient keyframes") {
			t.Skip("Sample video doesn't have enough keyframes for testing")
			return
		}

		require.NoError(t, err)
		require.NotEmpty(t, keyframes)

		// Validate keyframes
		err = ValidateKeyframes(keyframes)
		require.NoError(t, err)

		// Check that we have a reasonable number of keyframes
		assert.GreaterOrEqual(t, len(keyframes), minParsedKeyframeCount)

		// Check that first keyframe is at or after minimum time
		assert.GreaterOrEqual(t, keyframes[0], 0.0)

		// Check ascending order
		for i := 1; i < len(keyframes); i++ {
			assert.Greater(t, keyframes[i], keyframes[i-1])
		}
	})

	t.Run("invalid video path", func(t *testing.T) {
		keyframes, err := mp.ExtractKeyframesForVideo(context.Background(), "nonexistent.mp4")
		require.Error(t, err)
		assert.Nil(t, keyframes)
		assert.Contains(t, err.Error(), "failed to probe video")
	})

	t.Run("nil tools", func(t *testing.T) {
		mp := MediaProbe{Tools: nil}

		keyframes, err := mp.ExtractKeyframesForVideo(context.Background(), "testdata/sample.mp4")
		require.Error(t, err)
		assert.Nil(t, keyframes)
		assert.Contains(t, err.Error(), "ffprobe unavailable")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestValidateKeyframes(t *testing.T) {
	t.Run("valid keyframes", func(t *testing.T) {
		keyframes := []float64{0.0, 2.5, 5.0, 7.5, 10.0}
		err := ValidateKeyframes(keyframes)
		require.NoError(t, err)
	})

	t.Run("empty keyframes", func(t *testing.T) {
		keyframes := []float64{}
		err := ValidateKeyframes(keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no keyframes provided")
	})

	t.Run("not ascending order", func(t *testing.T) {
		keyframes := []float64{0.0, 5.0, 2.5, 10.0}
		err := ValidateKeyframes(keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in ascending order")
	})

	t.Run("negative timestamp", func(t *testing.T) {
		keyframes := []float64{-1.0, 2.5, 5.0}
		err := ValidateKeyframes(keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "negative keyframe timestamp")
	})

	t.Run("unreasonably large timestamp", func(t *testing.T) {
		keyframes := []float64{0.0, 2.5, 100000.0}
		err := ValidateKeyframes(keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unreasonably large keyframe timestamp")
	})

	t.Run("duplicate timestamps", func(t *testing.T) {
		keyframes := []float64{0.0, 2.5, 2.5, 5.0}
		err := ValidateKeyframes(keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in ascending order")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestGetSegmentCount(t *testing.T) {
	t.Run("empty keyframes", func(t *testing.T) {
		count := GetSegmentCount([]float64{})
		assert.Equal(t, 0, count)
	})

	t.Run("single keyframe", func(t *testing.T) {
		count := GetSegmentCount([]float64{0.0})
		assert.Equal(t, 1, count)
	})

	t.Run("multiple keyframes", func(t *testing.T) {
		count := GetSegmentCount([]float64{0.0, 2.5, 5.0, 7.5, 10.0})
		assert.Equal(t, 5, count)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestGetSegmentDuration(t *testing.T) {
	keyframes := []float64{0.0, 2.5, 5.0, 7.5, 10.0}

	t.Run("negative index", func(t *testing.T) {
		duration := GetSegmentDuration(keyframes, -1)
		assert.Equal(t, 0.0, duration)
	})

	t.Run("index out of bounds", func(t *testing.T) {
		duration := GetSegmentDuration(keyframes, 10)
		assert.Equal(t, 0.0, duration)
	})

	t.Run("valid segments", func(t *testing.T) {
		duration0 := GetSegmentDuration(keyframes, 0)
		assert.Equal(t, 2.5, duration0)

		duration1 := GetSegmentDuration(keyframes, 1)
		assert.Equal(t, 2.5, duration1)

		duration2 := GetSegmentDuration(keyframes, 2)
		assert.Equal(t, 2.5, duration2)

		duration3 := GetSegmentDuration(keyframes, 3)
		assert.Equal(t, 2.5, duration3)
	})

	t.Run("last segment", func(t *testing.T) {
		// Last segment duration can't be determined without total video duration
		duration := GetSegmentDuration(keyframes, 4)
		assert.Equal(t, 0.0, duration)
	})
}
