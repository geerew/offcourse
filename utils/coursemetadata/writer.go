package coursemetadata

import (
	"encoding/json"
	"path/filepath"
	"sync"

	"github.com/geerew/off-course/utils/concurrency"
	"github.com/geerew/off-course/utils/logger"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MetadataWriter handles sequential writes per course while allowing concurrent
// writes across courses
type MetadataWriter struct {
	fs     afero.Fs
	logger *logger.Logger

	// Per-course mutexes to ensure sequential writes
	mutexes concurrency.Map[string, *sync.Mutex]
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewMetadataWriter creates a new metadata writer
func NewMetadataWriter(fs afero.Fs, logger *logger.Logger) *MetadataWriter {
	return &MetadataWriter{
		fs:      fs,
		logger:  logger,
		mutexes: concurrency.NewMap[string, *sync.Mutex](),
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WriteMetadataAsync queues a write operation for the given course.
// Returns immediately without waiting for the file write to complete.
// Writes for the same course are processed sequentially.
func (w *MetadataWriter) WriteMetadataAsync(courseID, coursePath string, metadata *CourseMetadata) {
	if metadata == nil {
		return
	}

	// Get or create mutex for this course
	mutex, _ := w.mutexes.GetOrSet(courseID, &sync.Mutex{})

	// Start async write in goroutine
	go func() {
		mutex.Lock()
		defer mutex.Unlock()

		if err := w.writeMetadataAtomic(coursePath, metadata); err != nil {
			w.logger.Error().
				Err(err).
				Str("course_id", courseID).
				Str("course_path", coursePath).
				Msg("Failed to write course metadata")
		} else {
			w.logger.Debug().
				Str("course_id", courseID).
				Str("course_path", coursePath).
				Msg("Successfully wrote course metadata")
		}
	}()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// writeMetadataAtomic writes metadata using atomic file operations (temp file + rename)
// This helps handle cases where someone is manually editing the file
func (w *MetadataWriter) writeMetadataAtomic(coursePath string, metadata *CourseMetadata) error {
	metadataPath := filepath.Join(coursePath, MetadataFileName)
	tempPath := metadataPath + ".tmp"

	// Ensure directory exists
	dir := filepath.Dir(metadataPath)
	if err := w.fs.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Marshal JSON with indentation
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	// Add newline at end of file
	data = append(data, '\n')

	// Write to temp file first
	if err := afero.WriteFile(w.fs, tempPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename (overwrites existing file)
	// On most filesystems, rename is atomic
	if err := w.fs.Rename(tempPath, metadataPath); err != nil {
		// Clean up temp file on error
		_ = w.fs.Remove(tempPath)
		return err
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WriteMetadataSync writes metadata synchronously (for use during scans)
// Uses per-course mutex to ensure sequential writes
func (w *MetadataWriter) WriteMetadataSync(courseID, coursePath string, metadata *CourseMetadata) error {
	if metadata == nil {
		return nil
	}

	// Get or create mutex for this course
	mutex, _ := w.mutexes.GetOrSet(courseID, &sync.Mutex{})

	mutex.Lock()
	defer mutex.Unlock()

	return w.writeMetadataAtomic(coursePath, metadata)
}
