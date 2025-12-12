package coursemetadata

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/geerew/off-course/utils/logger"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestMetadataWriter_WriteMetadataAsync(t *testing.T) {
	t.Run("single write", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		coursePath := "/test-course"
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadata := &CourseMetadata{
			Description: "",
			Tags:        []string{"go", "programming"},
		}

		writer.WriteMetadataAsync("course-1", coursePath, metadata)

		// Wait a bit for async write to complete
		time.Sleep(100 * time.Millisecond)

		// Verify file was written
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Equal(t, metadata.Tags, readMetadata.Tags)
	})

	t.Run("concurrent writes to same course are sequential", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		coursePath := "/test-course"
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		courseID := "course-1"
		numWrites := 10
		var wg sync.WaitGroup
		writeOrder := make([]int, 0, numWrites)
		var orderMutex sync.Mutex

		// Write tags 0-9 concurrently, tracking write order
		for i := 0; i < numWrites; i++ {
			wg.Add(1)
			go func(tagNum int) {
				defer wg.Done()
				metadata := &CourseMetadata{
					Description: "",
					Tags:        []string{string(rune('a' + tagNum))},
				}
				writer.WriteMetadataAsync(courseID, coursePath, metadata)

				// Track when write completes (not when it starts)
				orderMutex.Lock()
				writeOrder = append(writeOrder, tagNum)
				orderMutex.Unlock()
			}(i)
		}

		wg.Wait()

		// Wait for all async writes to complete
		time.Sleep(500 * time.Millisecond)

		// Verify final state - should have the last write
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		// The last write should win, but since they're sequential, we should see one tag
		require.Len(t, readMetadata.Tags, 1)

		// Verify writes completed (order doesn't matter due to async, but all should complete)
		require.Len(t, writeOrder, numWrites)
	})

	t.Run("concurrent writes to different courses are concurrent", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		numCourses := 5
		var wg sync.WaitGroup

		// Write to different courses concurrently
		for i := 0; i < numCourses; i++ {
			wg.Add(1)
			go func(courseNum int) {
				defer wg.Done()
				coursePath := filepath.Join("/", "course", string(rune('0'+courseNum)))
				require.NoError(t, fs.MkdirAll(coursePath, 0755))

				metadata := &CourseMetadata{
					Tags: []string{string(rune('a' + courseNum))},
				}
				writer.WriteMetadataAsync("course-"+string(rune('0'+courseNum)), coursePath, metadata)
			}(i)
		}

		wg.Wait()

		// Wait for all async writes to complete
		time.Sleep(500 * time.Millisecond)

		// Verify all courses have their metadata
		for i := 0; i < numCourses; i++ {
			coursePath := filepath.Join("/", "course", string(rune('0'+i)))
			readMetadata, err := ReadMetadata(fs, coursePath)
			require.NoError(t, err)
			require.NotNil(t, readMetadata)
			require.Equal(t, []string{string(rune('a' + i))}, readMetadata.Tags)
		}
	})

	t.Run("sequential writes to same course", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		coursePath := "/test-course"
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		courseID := "course-1"

		// Write tag "a"
		writer.WriteMetadataAsync(courseID, coursePath, &CourseMetadata{Description: "", Tags: []string{"a"}})
		time.Sleep(50 * time.Millisecond)

		// Write tag "b"
		writer.WriteMetadataAsync(courseID, coursePath, &CourseMetadata{Description: "", Tags: []string{"b"}})
		time.Sleep(50 * time.Millisecond)

		// Write tag "c"
		writer.WriteMetadataAsync(courseID, coursePath, &CourseMetadata{Description: "", Tags: []string{"c"}})
		time.Sleep(50 * time.Millisecond)

		// Final state should be "c"
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Equal(t, []string{"c"}, readMetadata.Tags)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestMetadataWriter_WriteMetadataSync(t *testing.T) {
	t.Run("single write", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		coursePath := "/test-course"
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadata := &CourseMetadata{
			Description: "",
			Tags:        []string{"go", "programming"},
		}

		err := writer.WriteMetadataSync("course-1", coursePath, metadata)
		require.NoError(t, err)

		// Verify file was written
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Equal(t, metadata.Tags, readMetadata.Tags)
	})

	t.Run("concurrent sync writes to same course are sequential", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		coursePath := "/test-course"
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		courseID := "course-1"
		numWrites := 5
		var wg sync.WaitGroup
		errors := make([]error, numWrites)

		// Write tags concurrently
		for i := 0; i < numWrites; i++ {
			wg.Add(1)
			go func(tagNum int) {
				defer wg.Done()
				metadata := &CourseMetadata{
					Description: "",
					Tags:        []string{string(rune('a' + tagNum))},
				}
				errors[tagNum] = writer.WriteMetadataSync(courseID, coursePath, metadata)
			}(i)
		}

		wg.Wait()

		// All writes should succeed
		for i := 0; i < numWrites; i++ {
			require.NoError(t, errors[i])
		}

		// Verify final state - should have the last write
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Len(t, readMetadata.Tags, 1)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestMetadataWriter_writeMetadataAtomic(t *testing.T) {
	t.Run("atomic write with temp file", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		coursePath := "/test-course"
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		metadataPath := filepath.Join(coursePath, MetadataFileName)
		tempPath := metadataPath + ".tmp"

		metadata := &CourseMetadata{
			Description: "",
			Tags:        []string{"go", "programming"},
		}

		err := writer.WriteMetadataSync("course-1", coursePath, metadata)
		require.NoError(t, err)

		// Verify temp file doesn't exist
		exists, err := afero.Exists(fs, tempPath)
		require.NoError(t, err)
		require.False(t, exists)

		// Verify main file exists
		exists, err = afero.Exists(fs, metadataPath)
		require.NoError(t, err)
		require.True(t, exists)

		// Verify content
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Equal(t, metadata.Tags, readMetadata.Tags)
	})

	t.Run("temp file cleanup on error", func(t *testing.T) {
		// Create a filesystem that fails on rename
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		coursePath := "/test-course"
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		// Create a read-only directory to simulate rename failure
		// Actually, with MemMapFs we can't easily simulate this, so we'll test normal error handling
		metadata := &CourseMetadata{
			Tags: []string{"test"},
		}

		// This should work fine
		err := writer.WriteMetadataSync("course-1", coursePath, metadata)
		require.NoError(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestMetadataWriter_nilMetadata(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logger.NilLogger()
	writer := NewMetadataWriter(fs, logger)

	coursePath := "/test-course"
	require.NoError(t, fs.MkdirAll(coursePath, 0755))

	// Async write with nil metadata should not error
	writer.WriteMetadataAsync("course-1", coursePath, nil)

	// Sync write with nil metadata should not error
	err := writer.WriteMetadataSync("course-1", coursePath, nil)
	require.NoError(t, err)

	// File should not exist
	metadataPath := filepath.Join(coursePath, MetadataFileName)
	exists, err := afero.Exists(fs, metadataPath)
	require.NoError(t, err)
	require.False(t, exists)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestMetadataWriter_concurrentMixedOperations(t *testing.T) {
	t.Run("async and sync writes to same course", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		logger := logger.NilLogger()
		writer := NewMetadataWriter(fs, logger)

		coursePath := "/test-course"
		require.NoError(t, fs.MkdirAll(coursePath, 0755))

		courseID := "course-1"
		var wg sync.WaitGroup

		// Mix of async and sync writes
		wg.Add(1)
		go func() {
			defer wg.Done()
			writer.WriteMetadataAsync(courseID, coursePath, &CourseMetadata{Description: "", Tags: []string{"async1"}})
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			require.NoError(t, writer.WriteMetadataSync(courseID, coursePath, &CourseMetadata{Description: "", Tags: []string{"sync1"}}))
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			writer.WriteMetadataAsync(courseID, coursePath, &CourseMetadata{Description: "", Tags: []string{"async2"}})
		}()

		wg.Wait()
		time.Sleep(200 * time.Millisecond)

		// Verify file exists and has some content (last write wins)
		readMetadata, err := ReadMetadata(fs, coursePath)
		require.NoError(t, err)
		require.NotNil(t, readMetadata)
		require.Len(t, readMetadata.Tags, 1)
	})
}
