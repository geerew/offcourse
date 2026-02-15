package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DbWriter implements io.Writer for writing logs to the database with batching
type DbWriter struct {
	createLogsBatchFn DbWriterBatchFunc
	ctx               context.Context

	// Batching configuration
	batchSize     int
	flushInterval time.Duration

	// Batching state
	buffer      []*models.Log
	mu          sync.Mutex
	flushTicker *time.Ticker
	done        chan struct{}
	wg          sync.WaitGroup
	closed      bool
	closeMu     sync.Mutex
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DbWriterConfig holds configuration for DbWriter
type DbWriterConfig struct {
	// BatchSize is the number of logs to accumulate before flushing (default: 100)
	BatchSize int

	// FlushInterval is how often to flush logs even if batch isn't full (default: 5s)
	FlushInterval time.Duration
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewDbWriter creates a new DbWriter with batching support
func NewDbWriter(createLogsBatchFn DbWriterBatchFunc, config *DbWriterConfig) *DbWriter {
	if config == nil {
		config = &DbWriterConfig{
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

	dw := &DbWriter{
		createLogsBatchFn: createLogsBatchFn,
		ctx:               context.Background(),
		batchSize:         config.BatchSize,
		flushInterval:     config.FlushInterval,
		buffer:            make([]*models.Log, 0, config.BatchSize),
		done:              make(chan struct{}),
	}

	// Start periodic flush ticker
	dw.flushTicker = time.NewTicker(config.FlushInterval)
	dw.wg.Add(1)
	go dw.flushLoop()

	return dw
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Write implements io.Writer interface
// It parses zerolog JSON output and adds to the batch buffer
func (w *DbWriter) Write(p []byte) (n int, err error) {
	// Parse the zerolog JSON output
	var logEntry struct {
		Level     string                 `json:"level"`
		Time      time.Time              `json:"time"`
		Message   string                 `json:"message"`
		Component string                 `json:"component,omitempty"`
		Data      map[string]interface{} `json:"-"`
	}

	// Parse JSON, ignoring unknown fields
	if err := json.Unmarshal(p, &logEntry); err != nil {
		// If parsing fails, create a basic log entry
		logEntry = struct {
			Level     string                 `json:"level"`
			Time      time.Time              `json:"time"`
			Message   string                 `json:"message"`
			Component string                 `json:"component,omitempty"`
			Data      map[string]interface{} `json:"-"`
		}{
			Level:   "info",
			Time:    time.Now(),
			Message: string(p),
		}
	}

	// Extract additional fields from the JSON
	var rawData map[string]interface{}
	if err := json.Unmarshal(p, &rawData); err == nil {
		// Remove standard fields to get custom data
		delete(rawData, "level")
		delete(rawData, "time")
		delete(rawData, "message")
		delete(rawData, "component")
		logEntry.Data = rawData
	}

	// Convert zerolog level to our level
	var level int
	switch logEntry.Level {
	case "debug":
		level = int(LevelDebug)
	case "info":
		level = int(LevelInfo)
	case "warn":
		level = int(LevelWarn)
	case "error":
		level = int(LevelError)
	default:
		level = int(LevelInfo)
	}

	// Create log model
	log := &models.Log{
		Level:   level,
		Message: logEntry.Message,
		Data:    types.JsonMap(logEntry.Data),
	}

	// Add component to data if present
	if logEntry.Component != "" {
		if log.Data == nil {
			log.Data = make(types.JsonMap)
		}
		log.Data["component"] = logEntry.Component
	}

	// Add to buffer (thread-safe)
	w.mu.Lock()
	w.buffer = append(w.buffer, log)
	shouldFlush := len(w.buffer) >= w.batchSize
	w.mu.Unlock()

	// Flush if batch is full
	if shouldFlush {
		w.flush()
	}

	return len(p), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// flushLoop runs in a goroutine to periodically flush logs
func (w *DbWriter) flushLoop() {
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

// flush writes buffered logs to the database
func (w *DbWriter) flush() {
	w.mu.Lock()
	if len(w.buffer) == 0 {
		w.mu.Unlock()
		return
	}

	// Copy buffer and clear it
	logs := make([]*models.Log, len(w.buffer))
	copy(logs, w.buffer)
	w.buffer = w.buffer[:0]
	w.mu.Unlock()

	// Write to database
	if w.createLogsBatchFn != nil && len(logs) > 0 {
		if err := w.createLogsBatchFn(w.ctx, logs); err != nil {
			// If database write fails, we can't return an error from Write()
			// as it would break the logger. Log to stderr instead.
			fmt.Fprintf(io.Discard, "Failed to write logs to database: %v\n", err)
		}
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Close flushes any remaining logs and stops the flush ticker
func (w *DbWriter) Close() error {
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
	w.flush() // Final flush
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DbWriterBatchFunc is a function type for creating log entries in the database
type DbWriterBatchFunc func(context.Context, []*models.Log) error

