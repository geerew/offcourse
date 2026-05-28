package logger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestBatchWriter(t *testing.T) {
	// Test successfully flushing accumulated items via onFlush
	t.Run("flush on batch size", func(t *testing.T) {
		var batches [][]string
		w := NewBatchWriter(func(_ context.Context, batch []string) error {
			batches = append(batches, batch)
			return nil
		}, &BatchWriterConfig{
			BatchSize:     2,
			FlushInterval: 10 * time.Second,
		})

		w.Append("a")
		w.Append("b")
		require.NoError(t, w.Close())

		require.Len(t, batches, 1)
		require.Equal(t, []string{"a", "b"}, batches[0])
	})
}
