package logger

import (
	"io"

	"github.com/rs/zerolog"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NilLogger creates a logger that discards all output (useful for tests)
func NilLogger() *Logger {
	zlog := zerolog.New(io.Discard)
	return &Logger{
		zlog: zlog,
	}
}
