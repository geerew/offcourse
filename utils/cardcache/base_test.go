package cardcache

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"github.com/geerew/off-course/utils/filesystem"
	"github.com/geerew/off-course/utils/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestNew(t *testing.T) {
	// Test successfully creating a new card cache
	t.Run("create cache directory", func(t *testing.T) {
		fs := filesystem.New(afero.NewMemMapFs())
		testLogger := logger.NilLogger()

		cache, err := New(&CardCacheConfig{
			CachePath: "/test",
			FS:   fs,
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
	t.Run("fallback card", func(t *testing.T) {
		fs := filesystem.New(afero.NewMemMapFs())
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			FS:   fs,
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
	t.Run("nil config", func(t *testing.T) {
		_, err := New(nil)
		require.Error(t, err)
	})

	// Test erroring when the cache path is empty
	t.Run("empty cache path", func(t *testing.T) {
		_, err := New(&CardCacheConfig{
			FS:  filesystem.New(afero.NewMemMapFs()),
			Logger: logger.NilLogger(),
		})
		require.Error(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestOptimizeCard(t *testing.T) {
	// Test successfully generating an optimized card from a JPEG image
	t.Run("generates optimized card", func(t *testing.T) {
		fs := filesystem.New(afero.NewMemMapFs())
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			FS:   fs,
			Logger:    logger.NilLogger(),
		})
		require.NoError(t, err)

		testImagePath := filepath.Join(tmpDir, "test.jpg")
		writeTestJPEG(t, fs, testImagePath, 1200, 800)

		courseID := "testcourse"
		outputPath, err := cache.optimizedCardPath(courseID)
		require.NoError(t, err)

		require.NoError(t, cache.OptimizeCard(context.Background(), courseID, testImagePath, "jpeg-hash"))

		serve, err := cache.Get(courseID)
		require.NoError(t, err)
		require.Equal(t, outputPath, serve.Path)

		exists, err := cache.cardExists(outputPath)
		require.NoError(t, err)
		require.True(t, exists)

		optimizedInfo, err := fs.Stat(outputPath)
		require.NoError(t, err)
		originalInfo, err := fs.Stat(testImagePath)
		require.NoError(t, err)
		require.Less(t, optimizedInfo.Size(), originalInfo.Size())
		require.Equal(t, ".webp", filepath.Ext(outputPath))
	})

	// Test erroring when the image data is invalid
	t.Run("invalid image data", func(t *testing.T) {
		fs := filesystem.New(afero.NewMemMapFs())
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			FS:   fs,
			Logger:    logger.NilLogger(),
		})
		require.NoError(t, err)

		badPath := filepath.Join(tmpDir, "bad.jpg")
		require.NoError(t, afero.WriteFile(fs, badPath, []byte("not an image"), os.ModePerm))

		err = cache.OptimizeCard(context.Background(), "testcourse", badPath, "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to encode card")
	})

	// Test erroring when the context is cancelled
	t.Run("context cancellation", func(t *testing.T) {
		fs := filesystem.New(afero.NewMemMapFs())
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			FS:   fs,
			Logger:    logger.NilLogger(),
		})
		require.NoError(t, err)

		testImagePath := filepath.Join(tmpDir, "test.jpg")
		writeTestJPEG(t, fs, testImagePath, 100, 100)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = cache.OptimizeCard(ctx, "testcourse", testImagePath, "")
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDelete(t *testing.T) {
	// Test successfully deleting an existing card
	t.Run("success", func(t *testing.T) {
		fs := filesystem.New(afero.NewOsFs())
		testLogger := logger.NilLogger()
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			FS:   fs,
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
	t.Run("non-existent card", func(t *testing.T) {
		fs := filesystem.New(afero.NewOsFs())
		testLogger := logger.NilLogger()
		tmpDir := t.TempDir()

		cache, err := New(&CardCacheConfig{
			CachePath: tmpDir,
			FS:   fs,
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
	fs := filesystem.New(afero.NewMemMapFs())
	testLogger := logger.NilLogger()

	cache, err := New(&CardCacheConfig{
		CachePath: "/test",
		FS:   fs,
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

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func writeTestJPEG(t *testing.T, fs afero.Fs, path string, width, height int) {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: 128,
				A: 255,
			})
		}
	}

	require.NoError(t, fs.MkdirAll(filepath.Dir(path), os.ModePerm))

	f, err := fs.Create(path)
	require.NoError(t, err)
	require.NoError(t, jpeg.Encode(f, img, &jpeg.Options{Quality: 90}))
	require.NoError(t, f.Close())
}
