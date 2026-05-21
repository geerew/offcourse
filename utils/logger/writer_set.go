package logger

import (
	"io"
	"sync"

	"github.com/rs/zerolog"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// writerSet fans out zerolog output to multiple writers . This allows adding additional
// writers after New
type writerSet struct {
	mu      sync.RWMutex
	writers []io.Writer
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Write implements io.Writer
func (s *writerSet) Write(p []byte) (int, error) {
	return s.WriteLevel(zerolog.InfoLevel, p)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WriteLevel implements zerolog.LevelWriter
func (s *writerSet) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	s.mu.RLock()
	writers := s.writers
	s.mu.RUnlock()

	if len(writers) == 0 {
		return len(p), nil
	}

	for _, w := range writers {
		writeLevel(w, level, p)
	}

	return len(p), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// writeLevel forwards a log line to an io.Writer
func writeLevel(w io.Writer, level zerolog.Level, p []byte) {
	if lw, ok := w.(zerolog.LevelWriter); ok {
		_, _ = lw.WriteLevel(level, p)
		return
	}

	_, _ = w.Write(p)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// add appends a writer to the set
func (s *writerSet) add(w io.Writer) {
	s.mu.Lock()
	s.writers = append(s.writers, w)
	s.mu.Unlock()
}
