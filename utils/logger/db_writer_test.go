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

func Test_NewDbWriter(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, nil)

		require.NotNil(t, writer)
		// Test that it works with default config by writing and closing
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	})

	t.Run("custom config", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		config := &DbWriterConfig{
			BatchSize:     50,
			FlushInterval: 2 * time.Second,
		}
		writer := NewDbWriter(mock.CreateLogsBatch, config)

		require.NotNil(t, writer)
		// Test that it works with custom config
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	})

	t.Run("invalid batch size", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		config := &DbWriterConfig{
			BatchSize:     -1,
			FlushInterval: 1 * time.Second,
		}
		writer := NewDbWriter(mock.CreateLogsBatch, config)

		require.NotNil(t, writer)
		// Test that it works even with invalid batch size (should use default)
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	})

	t.Run("invalid flush interval", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		config := &DbWriterConfig{
			BatchSize:     50,
			FlushInterval: -1 * time.Second,
		}
		writer := NewDbWriter(mock.CreateLogsBatch, config)

		require.NotNil(t, writer)
		// Test that it works even with invalid flush interval (should use default)
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err)
		require.NoError(t, writer.Close())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DbWriter_Write(t *testing.T) {
	t.Run("parse valid JSON", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
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
		require.Equal(t, int(LevelInfo), logs[0].Level)
	})

	t.Run("parse invalid JSON", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     10,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		invalidJSON := `invalid json`
		n, err := writer.Write([]byte(invalidJSON))

		require.NoError(t, err) // Write should not return error
		require.Equal(t, len(invalidJSON), n)

		// Close to flush and verify
		require.NoError(t, writer.Close())
		logs := mock.GetLogs()
		require.Len(t, logs, 1)
		require.Equal(t, invalidJSON, logs[0].Message) // Should use raw string as message
	})

	t.Run("parse with component", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
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

	t.Run("parse with additional fields", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
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
		require.Equal(t, int(LevelError), logs[0].Level)
	})

	t.Run("level mapping", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     10,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		testCases := []struct {
			level     string
			expected  int
			jsonLevel string
		}{
			{"debug", int(LevelDebug), "debug"},
			{"info", int(LevelInfo), "info"},
			{"warn", int(LevelWarn), "warn"},
			{"error", int(LevelError), "error"},
			{"unknown", int(LevelInfo), "unknown"}, // Unknown defaults to Info
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

func Test_DbWriter_Batching(t *testing.T) {
	t.Run("batch size flush", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     5,
			FlushInterval: 10 * time.Second, // Long interval to avoid time-based flush
		})
		defer writer.Close()

		// Write 5 logs (exactly batch size)
		for i := 0; i < 5; i++ {
			logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"log ` + fmt.Sprintf("%d", i) + `"}`
			_, err := writer.Write([]byte(logJSON))
			require.NoError(t, err)
		}

		// Wait a bit for async flush
		time.Sleep(100 * time.Millisecond)

		// Should have flushed once
		require.Equal(t, 1, mock.GetCallCount())
		logs := mock.GetLogs()
		require.Len(t, logs, 5)
	})

	t.Run("buffer accumulation", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
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

	t.Run("multiple batches", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
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
		require.GreaterOrEqual(t, len(logs), 9) // At least 9 logs flushed

		// Close to flush remaining
		require.NoError(t, writer.Close())
		logs = mock.GetLogs()
		require.Len(t, logs, 10)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DbWriter_FlushInterval(t *testing.T) {
	t.Run("time-based flush", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     100, // Large batch size
			FlushInterval: 100 * time.Millisecond,
		})
		defer writer.Close()

		// Write 1 log (less than batch size)
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

func Test_DbWriter_Close(t *testing.T) {
	t.Run("flush on close", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     10,
			FlushInterval: 10 * time.Second,
		})

		// Write 3 logs that won't trigger auto-flush
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

	t.Run("double close", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     10,
			FlushInterval: 10 * time.Second,
		})

		require.NoError(t, writer.Close())
		require.NoError(t, writer.Close()) // Should not panic
	})

	t.Run("empty buffer close", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     10,
			FlushInterval: 10 * time.Second,
		})

		// Close without writing anything
		require.NoError(t, writer.Close())
		require.Equal(t, 0, mock.GetCallCount())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DbWriter_Concurrency(t *testing.T) {
	t.Run("concurrent writes", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     50,
			FlushInterval: 1 * time.Second,
		})
		defer writer.Close()

		// Write concurrently
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

		// Close to flush all
		require.NoError(t, writer.Close())

		logs := mock.GetLogs()
		require.Len(t, logs, numWriters*logsPerWriter)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_NewDbWriter_WithDAO(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     10,
			FlushInterval: 1 * time.Second,
		})

		require.NotNil(t, writer)
		require.NoError(t, writer.Close())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DbWriter_ErrorHandling(t *testing.T) {
	t.Run("database error", func(t *testing.T) {
		mock := mocks.NewMockBatchWriter()
		mock.SetShouldError(true, 0)

		writer := NewDbWriter(mock.CreateLogsBatch, &DbWriterConfig{
			BatchSize:     2,
			FlushInterval: 10 * time.Second,
		})
		defer writer.Close()

		// Write 2 logs to trigger flush
		logJSON := `{"level":"info","time":"2024-01-01T00:00:00Z","message":"test"}`
		_, err := writer.Write([]byte(logJSON))
		require.NoError(t, err) // Write itself should not error

		_, err = writer.Write([]byte(logJSON))
		require.NoError(t, err)

		// Wait for flush
		time.Sleep(100 * time.Millisecond)

		// Writer should still be functional (errors are swallowed)
		require.NotNil(t, writer)
	})
}
