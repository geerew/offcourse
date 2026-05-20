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

		cachePath, err := cache.optimizedCardPath("test")
		require.NoError(t, err)
		cachePath = filepath.Dir(cachePath)
		require.Contains(t, cachePath, "cards")
	})

	// Test successfully writing a fallback card
	t.Run("writes fallback card", func(t *testing.T) {
		appFs := appfs.New(afero.NewMemMapFs())
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			AppFs:     appFs,
			Logger:    logger.NilLogger(),
		})
		require.NoError(t, err)

		fallbackPath := cache.fallbackPath()
		exists, err := cache.cardExists(fallbackPath)
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, ".webp", filepath.Ext(fallbackPath))
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

func TestOptimizeCard(t *testing.T) {
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

		courseID := "testcourse"
		outputPath, err := cache.optimizedCardPath(courseID)
		require.NoError(t, err)
		ctx := context.Background()

		require.NoError(t, cache.OptimizeCard(ctx, courseID, testImagePath))

		serve, err := cache.Get(courseID)
		require.NoError(t, err)
		require.NotEmpty(t, serve.Path)

		exists, err := cache.cardExists(outputPath)
		require.NoError(t, err)
		if exists {
			optimizedInfo, err := appFs.Fs.Stat(outputPath)
			require.NoError(t, err)
			originalInfo, err := appFs.Fs.Stat(testImagePath)
			require.NoError(t, err)
			require.Less(t, optimizedInfo.Size(), originalInfo.Size())
			require.Equal(t, outputPath, serve.Path)
		} else {
			require.Equal(t, testImagePath, serve.Path)
		}
		require.Equal(t, ".webp", filepath.Ext(outputPath))
	})

	// Test erroring when FFmpeg is not configured
	t.Run("requires ffmpeg", func(t *testing.T) {
		cache := &CardCache{
			config:    &CardCacheConfig{Logger: logger.NilLogger()},
			cachePath: filepath.Join(t.TempDir(), "cards"),
		}

		err := cache.OptimizeCard(context.Background(), "testcourse", "in.jpg")
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

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = cache.OptimizeCard(ctx, "testcourse", testImagePath)
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDelete(t *testing.T) {
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

		err = cache.deleteCard(testCardPath)
		require.NoError(t, err)

		exists, err := cache.cardExists(testCardPath)
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
		err = cache.deleteCard(nonExistentPath)
		require.NoError(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestOptimizedPath(t *testing.T) {
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
		cardPath, err := cache.optimizedCardPath(courseID)
		require.NoError(t, err)
		require.Contains(t, cardPath, courseID)
		require.Equal(t, ".webp", filepath.Ext(cardPath))
	})

	// Test erroring when the course ID is empty
	t.Run("rejects empty course id", func(t *testing.T) {
		_, err := cache.optimizedCardPath("")
		require.ErrorIs(t, err, ErrInvalidCourseID)
	})

	// Test erroring when the course ID contains path traversal
	t.Run("rejects path traversal", func(t *testing.T) {
		_, err := cache.optimizedCardPath("../escape")
		require.ErrorIs(t, err, ErrInvalidCourseID)
	})

	// Test erroring when the course ID contains path separators
	t.Run("rejects path separators", func(t *testing.T) {
		_, err := cache.optimizedCardPath(`a/b`)
		require.ErrorIs(t, err, ErrInvalidCourseID)
	})
}
