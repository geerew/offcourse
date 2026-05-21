package logger

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/geerew/off-course/utils/mocks"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_NewLogBatchWriter(t *testing.T) {
	// Test successfully writing with default content
	t.Run("default config", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, nil)

		require.NotNil(t, writer)
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	})

	// Test successfully writing with custom config
	t.Run("custom config", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		config := &BatchWriterConfig{
			BatchSize:     50,
			FlushInterval: 2 * time.Second,
		}
		writer := NewLogBatchWriter(mock.CreateLogsBatch, config)

		require.NotNil(t, writer)
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	})

	// Test successfully writing when batch size is invalid
	t.Run("invalid batch size", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		config := &BatchWriterConfig{
			BatchSize:     -1,
			FlushInterval: 1 * time.Second,
		}
		writer := NewLogBatchWriter(mock.CreateLogsBatch, config)

		require.NotNil(t, writer)

		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	})

	// Test successfully writing when flush interval is invalid
	t.Run("invalid flush interval", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		config := &BatchWriterConfig{
			BatchSize:     50,
			FlushInterval: -1 * time.Second,
		}
		writer := NewLogBatchWriter(mock.CreateLogsBatch, config)

		require.NotNil(t, writer)
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogBatchWriter_Write(t *testing.T) {
	// Test successfully writing valid JSON
	t.Run("parse valid JSON", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test message"}`
		n, err := writer.Write([]byte(logJSON))

		require.NoError(t, err)
		require.Equal(t, len(logJSON), n)

		// Close to flush and verify
		require.NoError(t, writer.Close())
		logs := mock.GetLogs()
		require.Len(t, logs, 1)
		require.Equal(t, "test message", logs[0].Message)
		require.Equal(t, "info", logs[0].Level)
	})

	// Test successfully writing invalid JSON
	t.Run("parse invalid JSON", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		invalidJSON := `invalid json`
		n, err := writer.Write([]byte(invalidJSON))

		require.NoError(t, err)
		require.Equal(t, len(invalidJSON), n)

		// Close to flush and verify
		require.NoError(t, writer.Close())
		logs := mock.GetLogs()
		require.Len(t, logs, 1)
		require.Equal(t, invalidJSON, logs[0].Message)
	})

	// Test successfully writing with component
	t.Run("parse with component", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test","component":"api"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)

		require.NoError(t, writer.Close())
		logs := mock.GetLogs()
		require.Len(t, logs, 1)
		require.NotNil(t, logs[0].Data)
		require.Equal(t, "api", logs[0].Data["component"])
	})

	// Test successfully writing with additional fields
	t.Run("parse with additional fields", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		logJSON := `{"level":"error","time":"2024-01-01T00:00:00Z","message":"error occurred","error":"something went wrong","user_id":"123"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)

		require.NoError(t, writer.Close())
		logs := mock.GetLogs()
		require.Len(t, logs, 1)
		require.NotNil(t, logs[0].Data)
		require.Equal(t, "something went wrong", logs[0].Data["error"])
		require.Equal(t, "123", logs[0].Data["user_id"])
		require.Equal(t, "error", logs[0].Level)
	})

	// Test successfully writing with different levels
	t.Run("level mapping", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		testCases := []struct {
			level     string
			expected  string
			jsonLevel string
		}{
			{"debug", "debug", "debug"},
			{"info", "info", "info"},
			{"warn", "warn", "warn"},
			{"error", "error", "error"},
			{"unknown", "info", "unknown"},
		}

		for i, tc := range testCases {
			logJSON := `{"level":"` + tc.jsonLevel + `","time":"2024-01-01T00:00:00Z","message":"test ` + tc.level + `"}`
			_, err := writer.Write([]byte(logJSON))
			require.NoError(t, err, "test case %d", i)
		}

		require.NoError(t, writer.Close())
		logs := mock.GetLogs()
		require.Len(t, logs, len(testCases))

		for i, tc := range testCases {
			require.Equal(t, tc.expected, logs[i].Level, "test case %d: level %s", i, tc.level)
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogBatchWriter_Batching(t *testing.T) {
	// Test successfully writing when batch size is reached
	t.Run("batch size flush", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     5,
			FlushInterval: 10 * time.Second,
		})
		defer writer.Close()

		// Write 5 logs (same as the batch size)
		for i := 0; i < 5; i++ {
			logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"log ` + fmt.Sprintf("%d", i) + `"}`
			_, err := writer.Write([]byte(logJSON))
			require.NoError(t, err)
		}

		// Wait a bit for async flush
		time.Sleep(100 * time.Millisecond)

		require.Equal(t, 1, mock.GetCallCount())
		logs := mock.GetLogs()
		require.Len(t, logs, 5)
	})

	// Test successfully writing when buffer accumulation is reached
	t.Run("buffer accumulation", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 10 * time.Second,
		})
		defer writer.Close()

		// Write 3 logs (less than batch size)
		for i := 0; i < 3; i++ {
			logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"log ` + fmt.Sprintf("%d", i) + `"}`
			_, err := writer.Write([]byte(logJSON))
			require.NoError(t, err)
		}

		// Should not have flushed yet
		time.Sleep(100 * time.Millisecond)
		require.Equal(t, 0, mock.GetCallCount())

		// Close should flush remaining
		require.NoError(t, writer.Close())
		require.Equal(t, 1, mock.GetCallCount())
		logs := mock.GetLogs()
		require.Len(t, logs, 3)
	})

	// Test successfully writing when multiple batches are created
	t.Run("multiple batches", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     3,
			FlushInterval: 10 * time.Second,
		})
		defer writer.Close()

		// Write 10 logs (should create 3 full batches + 1 partial)
		for i := 0; i < 10; i++ {
			logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"log ` + fmt.Sprintf("%d", i) + `"}`
			_, err := writer.Write([]byte(logJSON))
			require.NoError(t, err)
		}

		// Wait for async flushes
		time.Sleep(200 * time.Millisecond)

		// Should have flushed 3 times (for 3 full batches)
		require.GreaterOrEqual(t, mock.GetCallCount(), 3)
		logs := mock.GetLogs()
		require.GreaterOrEqual(t, len(logs), 9)

		// Close to flush remaining
		require.NoError(t, writer.Close())
		logs = mock.GetLogs()
		require.Len(t, logs, 10)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogBatchWriter_FlushInterval(t *testing.T) {
	// Test successfully writing when flush interval is reached
	t.Run("time-based flush", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     100, // Large batch size
			FlushInterval: 100 * time.Millisecond,
		})
		defer writer.Close()

		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)

		// Should flush after interval
		time.Sleep(150 * time.Millisecond)

		require.Equal(t, 1, mock.GetCallCount())
		logs := mock.GetLogs()
		require.Len(t, logs, 1)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogBatchWriter_Close(t *testing.T) {
	// Test successfully flushing on close
	t.Run("flush on close", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 10 * time.Second,
		})

		for i := 0; i < 3; i++ {
			logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"log ` + fmt.Sprintf("%d", i) + `"}`
			_, err := writer.Write([]byte(logJSON))
			require.NoError(t, err)
		}

		// Verify not flushed yet
		time.Sleep(100 * time.Millisecond)
		require.Equal(t, 0, mock.GetCallCount())

		// Close should flush
		require.NoError(t, writer.Close())
		require.Equal(t, 1, mock.GetCallCount())
		logs := mock.GetLogs()
		require.Len(t, logs, 3)
	})

	// Test successfully closing twice
	t.Run("double close", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 10 * time.Second,
		})

		require.NoError(t, writer.Close())
		require.NoError(t, writer.Close())
	})

	// Test successfully closing when the buffer is empty
	t.Run("empty buffer close", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     10,
			FlushInterval: 10 * time.Second,
		})

		require.NoError(t, writer.Close())
		require.Equal(t, 0, mock.GetCallCount())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogBatchWriter_Concurrency(t *testing.T) {
	// Test successfully writing concurrently
	t.Run("concurrent writes", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     50,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		var wg sync.WaitGroup
		numWriters := 10
		logsPerWriter := 5

		for i := 0; i < numWriters; i++ {
			wg.Add(1)
			go func(writerID int) {
				defer wg.Done()
				for j := 0; j < logsPerWriter; j++ {
					logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"log ` + fmt.Sprintf("%d-%d", writerID, j) + `"}`
					_, err := writer.Write([]byte(logJSON))
					require.NoError(t, err)
				}
			}(i)
		}

		wg.Wait()

		require.NoError(t, writer.Close())

		logs := mock.GetLogs()
		require.Len(t, logs, numWriters*logsPerWriter)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogBatchWriter_ErrorHandling(t *testing.T) {
	// Test error due to onFlush returning an error on the first batch
	t.Run("flush error", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		mock.SetShouldError(true, 0)

		writer := NewLogBatchWriter(mock.CreateLogsBatch, &BatchWriterConfig{
			BatchSize:     2,
			FlushInterval: 10 * time.Second,
		})
		defer writer.Close()

		// Write 2 logs to trigger flush
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)

		_, err = writer.Write([]byte(logJSON))
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)
		require.NotNil(t, writer)
	})
}
