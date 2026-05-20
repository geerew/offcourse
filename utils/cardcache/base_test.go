package cardcache

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/media"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestNew(t *testing.T) {
	// Test successfully creating a new card cache
	t.Run("creates cache directory", func(t *testing.T) {
		appFs := appfs.New(afero.NewMemMapFs())
		testLogger := logger.NilLogger()

		cache, err := New(&CardCacheConfig{
			CachePath: "/test",
			AppFs:     appFs,
			Logger:    testLogger,
		})

		require.NoError(t, err)
		require.NotNil(t, cache)

		cachePath, err := cache.GetCardPath("test")
		require.NoError(t, err)
		cachePath = filepath.Dir(cachePath)
		require.Contains(t, cachePath, "cards")
	})

	// Test erroring when the config is nil
	t.Run("rejects nil config", func(t *testing.T) {
		_, err := New(nil)
		require.Error(t, err)
	})

	// Test erroring when the cache path is empty
	t.Run("rejects empty cache path", func(t *testing.T) {
		_, err := New(&CardCacheConfig{
			AppFs:  appfs.New(afero.NewMemMapFs()),
			Logger: logger.NilLogger(),
		})
		require.Error(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestEnsureFallbackCard(t *testing.T) {
	// Test successfully writing a fallback card
	t.Run("writes fallback card", func(t *testing.T) {
		appFs := appfs.New(afero.NewMemMapFs())
		testLogger := logger.NilLogger()
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			AppFs:     appFs,
			Logger:    testLogger,
		})
		require.NoError(t, err)

		fallbackPath := cache.GetFallbackPath()

		err = cache.EnsureFallbackCard(fallbackPath)
		require.NoError(t, err)

		exists, err := cache.CardExists(fallbackPath)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, ".webp", filepath.Ext(fallbackPath))
	})

	// Test idempotency of the fallback card
	t.Run("is idempotent", func(t *testing.T) {
		appFs := appfs.New(afero.NewMemMapFs())
		testLogger := logger.NilLogger()
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			AppFs:     appFs,
			Logger:    testLogger,
		})
		require.NoError(t, err)

		fallbackPath := cache.GetFallbackPath()

		require.NoError(t, cache.EnsureFallbackCard(fallbackPath))
		require.NoError(t, cache.EnsureFallbackCard(fallbackPath))

		exists, err := cache.CardExists(fallbackPath)
		require.NoError(t, err)
		require.True(t, exists)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestGenerateOptimizedCard(t *testing.T) {
	// Test successfully generating an optimized card from a JPEG image
	t.Run("generates optimized card from JPEG", func(t *testing.T) {
		appFs := appfs.New(afero.NewOsFs())
		testLogger := logger.NilLogger()

		ffmpeg, err := media.NewFFmpeg()
		if err != nil {
			t.Skip("FFmpeg not available for testing")
		}

		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			AppFs:     appFs,
			Logger:    testLogger,
			FFmpeg:    ffmpeg,
		})
		require.NoError(t, err)

		testImagePath := filepath.Join(tmpDir, "test.jpg")
		ffmpegPath := ffmpeg.GetFFmpegPath()
		createImageCmd := exec.Command(ffmpegPath,
			"-f", "lavfi",
			"-i", "color=c=red:s=100x100:d=1",
			"-frames:v", "1",
			"-y",
			testImagePath,
		)
		err = createImageCmd.Run()
		if err != nil {
			t.Skipf("Failed to create test image: %v", err)
		}

		outputPath := filepath.Join(tmpDir, "optimized.webp")
		ctx := context.Background()

		err = cache.GenerateOptimizedCard(ctx, testImagePath, outputPath)
		require.NoError(t, err)

		exists, err := cache.CardExists(outputPath)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, ".webp", filepath.Ext(outputPath))
	})

	// Test erroring when FFmpeg is not configured
	t.Run("requires ffmpeg", func(t *testing.T) {
		cache := &CardCache{config: &CardCacheConfig{Logger: logger.NilLogger()}}

		err := cache.GenerateOptimizedCard(context.Background(), "in.jpg", "out.webp")
		require.Error(t, err)
		require.Contains(t, err.Error(), "ffmpeg is not configured")
	})

	// Test successfully handling context cancellation
	t.Run("handles context cancellation", func(t *testing.T) {
		appFs := appfs.New(afero.NewOsFs())
		testLogger := logger.NilLogger()

		ffmpeg, err := media.NewFFmpeg()
		if err != nil {
			t.Skip("FFmpeg not available for testing")
		}

		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			AppFs:     appFs,
			Logger:    testLogger,
			FFmpeg:    ffmpeg,
		})
		require.NoError(t, err)

		testImagePath := filepath.Join(tmpDir, "test.jpg")
		ffmpegPath := ffmpeg.GetFFmpegPath()
		createImageCmd := exec.Command(ffmpegPath,
			"-f", "lavfi",
			"-i", "color=c=red:s=100x100:d=1",
			"-frames:v", "1",
			"-y",
			testImagePath,
		)
		err = createImageCmd.Run()
		if err != nil {
			t.Skipf("Failed to create test image: %v", err)
		}

		outputPath := filepath.Join(tmpDir, "optimized.webp")
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = cache.GenerateOptimizedCard(ctx, testImagePath, outputPath)
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDeleteCard(t *testing.T) {
	// Test successfully deleting an existing card
	t.Run("deletes existing card", func(t *testing.T) {
		appFs := appfs.New(afero.NewOsFs())
		testLogger := logger.NilLogger()
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			AppFs:     appFs,
			Logger:    testLogger,
		})
		require.NoError(t, err)

		testCardPath := filepath.Join(tmpDir, "test.webp")
		err = os.WriteFile(testCardPath, []byte("test"), 0o644)
		require.NoError(t, err)

		err = cache.DeleteCard(testCardPath)
		require.NoError(t, err)

		exists, err := cache.CardExists(testCardPath)
		require.NoError(t, err)
		require.False(t, exists)
	})

	// Test successfully handling a non-existent card
	t.Run("handles non-existent card gracefully", func(t *testing.T) {
		appFs := appfs.New(afero.NewOsFs())
		testLogger := logger.NilLogger()
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			AppFs:     appFs,
			Logger:    testLogger,
		})
		require.NoError(t, err)

		nonExistentPath := filepath.Join(tmpDir, "nonexistent.webp")
		err = cache.DeleteCard(nonExistentPath)
		require.NoError(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestGetCardPath(t *testing.T) {
	appFs := appfs.New(afero.NewMemMapFs())
	testLogger := logger.NilLogger()

	cache, err := New(&CardCacheConfig{
		CachePath: "/test",
		AppFs:     appFs,
		Logger:    testLogger,
	})
	require.NoError(t, err)

	// Test successfully getting a card path
	t.Run("returns correct path format", func(t *testing.T) {
		courseID := "test-course-id"
		cardPath, err := cache.GetCardPath(courseID)
		require.NoError(t, err)
		require.Contains(t, cardPath, courseID)
		require.Equal(t, ".webp", filepath.Ext(cardPath))
	})

	// Test erroring when the course ID is empty
	t.Run("rejects empty course id", func(t *testing.T) {
		_, err := cache.GetCardPath("")
		require.ErrorIs(t, err, ErrInvalidCourseID)
	})

	// Test erroring when the course ID contains path traversal
	t.Run("rejects path traversal", func(t *testing.T) {
		_, err := cache.GetCardPath("../escape")
		require.ErrorIs(t, err, ErrInvalidCourseID)
	})

	// Test erroring when the course ID contains path separators
	t.Run("rejects path separators", func(t *testing.T) {
		_, err := cache.GetCardPath(`a/b`)
		require.ErrorIs(t, err, ErrInvalidCourseID)
	})
}
