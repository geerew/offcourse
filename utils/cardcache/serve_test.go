package cardcache

import (
	"io"
	"os"
	"testing"

	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	appFs := appfs.New(afero.NewMemMapFs())
	tmpDir := t.TempDir()

	cache, err := New(&CardCacheConfig{
		CachePath: tmpDir,
		AppFs:     appFs,
		Logger:    logger.NilLogger(),
	})
	require.NoError(t, err)

	// Test successfully serving a fallback card following a cache miss
	t.Run("cache miss serves fallback", func(t *testing.T) {
		serve, err := cache.Get("unknown-course")
		require.NoError(t, err)
		require.True(t, serve.Fallback)
	})

	// Test successfully serving an optimized card from the index
	t.Run("optimized", func(t *testing.T) {
		courseID := "course-optimized"
		optimizedPath, err := cache.optimizedCardPath(courseID)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(appFs.Fs, optimizedPath, []byte("webp"), os.ModePerm))
		require.NoError(t, cache.setServeOptimized(courseID, "hash-opt"))

		serve, err := cache.Get(courseID)
		require.NoError(t, err)
		require.False(t, serve.Fallback)
		require.Equal(t, optimizedPath, serve.Path)
		require.Equal(t, "image/webp", serve.ContentType)
	})

	// Test successfully serving an original card when optimization skips the cache
	t.Run("original", func(t *testing.T) {
		courseID := "course-original"
		originalPath := "/course-1/card-original.png"
		require.NoError(t, appFs.Fs.MkdirAll("/course-1", os.ModePerm))
		require.NoError(t, afero.WriteFile(appFs.Fs, originalPath, []byte("original"), os.ModePerm))
		cache.setServeOriginal(courseID, originalPath, "hash-orig")

		serve, err := cache.Get(courseID)
		require.NoError(t, err)
		require.False(t, serve.Fallback)
		require.Equal(t, originalPath, serve.Path)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestWarm(t *testing.T) {
	appFs := appfs.New(afero.NewMemMapFs())
	tmpDir := t.TempDir()

	cache, err := New(&CardCacheConfig{
		CachePath: tmpDir,
		AppFs:     appFs,
		Logger:    logger.NilLogger(),
	})
	require.NoError(t, err)

	// Test successfully warming the card cache
	t.Run("picks optimized webp on disk", func(t *testing.T) {
		courseID := "course-warm-opt"
		optimizedPath, err := cache.optimizedCardPath(courseID)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(appFs.Fs, optimizedPath, []byte("webp"), os.ModePerm))

		warmed := cache.Warm([]CourseCardRef{{ID: courseID, CardPath: "/missing/original.png"}})
		require.Equal(t, 1, warmed)

		serve, err := cache.Get(courseID)
		require.NoError(t, err)
		require.False(t, serve.Fallback)
		require.Equal(t, optimizedPath, serve.Path)
	})

	// Test successfully warming the card cache with an original card when no optimized file
	// is present
	t.Run("picks original when no optimized file", func(t *testing.T) {
		courseID := "course-warm-orig"
		originalPath := "/course-1/card.png"
		require.NoError(t, appFs.Fs.MkdirAll("/course-1", os.ModePerm))
		require.NoError(t, afero.WriteFile(appFs.Fs, originalPath, []byte("original"), os.ModePerm))

		warmed := cache.Warm([]CourseCardRef{{ID: courseID, CardPath: originalPath}})
		require.Equal(t, 1, warmed)

		serve, err := cache.Get(courseID)
		require.NoError(t, err)
		require.False(t, serve.Fallback)
		require.Equal(t, originalPath, serve.Path)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully deleting an optimized card
func TestDeleteCardForCourse(t *testing.T) {
	appFs := appfs.New(afero.NewMemMapFs())
	tmpDir := t.TempDir()

	cache, err := New(&CardCacheConfig{
		CachePath: tmpDir,
		AppFs:     appFs,
		Logger:    logger.NilLogger(),
	})
	require.NoError(t, err)

	courseID := "course-xyz"
	optimizedPath, err := cache.optimizedCardPath(courseID)
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(appFs.Fs, optimizedPath, []byte("webp"), os.ModePerm))
	cache.setServeOriginal(courseID, "/card.png", "")

	require.NoError(t, cache.Delete(courseID))

	exists, err := cache.cardExists(optimizedPath)
	require.NoError(t, err)
	require.False(t, exists)

	serve, err := cache.Get(courseID)
	require.NoError(t, err)
	require.True(t, serve.Fallback)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestOpenCard(t *testing.T) {
	appFs := appfs.New(afero.NewMemMapFs())
	tmpDir := t.TempDir()

	cache, err := New(&CardCacheConfig{
		CachePath: tmpDir,
		AppFs:     appFs,
		Logger:    logger.NilLogger(),
	})
	require.NoError(t, err)

	courseID := "course-open"
	optimizedPath, err := cache.optimizedCardPath(courseID)
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(appFs.Fs, optimizedPath, []byte("webp-bytes"), os.ModePerm))
	require.NoError(t, cache.setServeOptimized(courseID, "open-hash"))

	rc, serve, err := cache.OpenCard(courseID)
	require.NoError(t, err)
	require.Equal(t, "open-hash", serve.CardHash)
	require.Equal(t, optimizedPath, serve.Path)

	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, "webp-bytes", string(data))
}
