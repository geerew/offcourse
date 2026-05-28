package mocks

import (
	"context"
	"sync"

	"github.com/geerew/off-course/models"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MockBatchWriter is a mock BatchFlushFunc target for logger tests
type MockBatchWriter struct {
	mu           sync.Mutex
	logs         []*models.Log
	callCount    int
	shouldError  bool
	errorOnBatch int // Which batch call should error (0-indexed)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewMockBatchWriter creates a new MockBatchWriter
func NewMockBatchWriter() *MockBatchWriter {
	return &MockBatchWriter{
		logs: make([]*models.Log, 0),
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateLogsBatch implements the batch writer interface
func (m *MockBatchWriter) CreateLogsBatch(ctx context.Context, logs []*models.Log) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount++
	if m.shouldError && m.callCount == m.errorOnBatch+1 {
		return &MockError{Message: "mock database error"}
	}

	m.logs = append(m.logs, logs...)
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetLogs returns all logs that have been written
func (m *MockBatchWriter) GetLogs() []*models.Log {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]*models.Log, len(m.logs))
	copy(result, m.logs)
	return result
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetCallCount returns the number of times CreateLogsBatch has been called
func (m *MockBatchWriter) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Reset clears all logs and resets counters
func (m *MockBatchWriter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = make([]*models.Log, 0)
	m.callCount = 0
	m.shouldError = false
	m.errorOnBatch = 0
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SetShouldError configures the mock to return an error
func (m *MockBatchWriter) SetShouldError(shouldError bool, errorOnBatch int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorOnBatch = errorOnBatch
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MockError is a mock error type
type MockError struct {
	Message string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Error implements the error interface
func (e *MockError) Error() string {
	return e.Message
}
