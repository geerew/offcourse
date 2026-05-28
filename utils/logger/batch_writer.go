package logger

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// BatchFlushFunc is called with each flushed batch of items
type BatchFlushFunc[T any] func(context.Context, []T) error

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// BatchWriter buffers items and calls onFlush when the batch is full or on interval
type BatchWriter[T any] struct {
	onFlush BatchFlushFunc[T]
	ctx     context.Context

	// batchSize and flushInterval control when the buffer is flushed (defaults in
	// BatchWriterConfig: 100 and 5s)
	batchSize     int
	flushInterval time.Duration

	// buffer holds items not yet flushed; mu guards concurrent Append calls
	buffer []T
	mu     sync.Mutex

	// flushTicker triggers periodic flush; done signals flushLoop to exit; wg waits for it
	flushTicker *time.Ticker
	done        chan struct{}
	wg          sync.WaitGroup

	// closed and closeMu make Close idempotent
	closed  bool
	closeMu sync.Mutex
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// BatchWriterConfig holds configuration for BatchWriter
type BatchWriterConfig struct {
	// BatchSize is the number of items to accumulate before flushing (default: 100)
	BatchSize int

	// FlushInterval is how often to flush even if the batch is not full (default: 5s)
	FlushInterval time.Duration
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewBatchWriter creates a BatchWriter that calls onFlush with each batch
func NewBatchWriter[T any](onFlush BatchFlushFunc[T], config *BatchWriterConfig) *BatchWriter[T] {
	if config == nil {
		config = &BatchWriterConfig{
			BatchSize:     100,
			FlushInterval: 5 * time.Second,
		}
	}

	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}

	if config.FlushInterval <= 0 {
		config.FlushInterval = 5 * time.Second
	}

	w := &BatchWriter[T]{
		onFlush:       onFlush,
		ctx:           context.Background(),
		batchSize:     config.BatchSize,
		flushInterval: config.FlushInterval,
		buffer:        make([]T, 0, config.BatchSize),
		done:          make(chan struct{}),
	}

	w.flushTicker = time.NewTicker(config.FlushInterval)
	w.wg.Add(1)
	go w.flushLoop()

	return w
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Append adds an item to the buffer and flushes when batch size is reached
func (w *BatchWriter[T]) Append(item T) {
	w.mu.Lock()
	w.buffer = append(w.buffer, item)
	shouldFlush := len(w.buffer) >= w.batchSize
	w.mu.Unlock()

	if shouldFlush {
		w.flush()
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// flushLoop runs in a goroutine to periodically flush the buffer
func (w *BatchWriter[T]) flushLoop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.flushTicker.C:
			w.flush()
		case <-w.done:
			return
		}
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// flush delivers the current buffer to onFlush
func (w *BatchWriter[T]) flush() {
	w.mu.Lock()
	if len(w.buffer) == 0 {
		w.mu.Unlock()
		return
	}

	batch := make([]T, len(w.buffer))
	copy(batch, w.buffer)
	w.buffer = w.buffer[:0]
	w.mu.Unlock()

	if w.onFlush != nil && len(batch) > 0 {
		if err := w.onFlush(w.ctx, batch); err != nil {
			fmt.Fprintf(os.Stderr, "failed to flush batch: %v\n", err)
		}
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Close flushes any remaining items and stops the flush ticker
func (w *BatchWriter[T]) Close() error {
	w.closeMu.Lock()
	if w.closed {
		w.closeMu.Unlock()
		return nil
	}
	w.closed = true
	w.closeMu.Unlock()

	if w.flushTicker != nil {
		w.flushTicker.Stop()
	}
	close(w.done)
	w.wg.Wait()
	w.flush()

	return nil
}
