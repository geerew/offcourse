package logger

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// LogLevel represents the log level
type LogLevel int

const (
	LevelDebug LogLevel = -1
	LevelInfo  LogLevel = 0
	LevelWarn  LogLevel = 2
	LevelError LogLevel = 1
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// LoggerConfig holds the logger configuration
type LoggerConfig struct {
	// Level sets the minimum log level (Debug, Info, Warn, Error)
	Level LogLevel

	// ConsoleOutput enables pretty console output
	ConsoleOutput bool

	// AdditionalWriters are extra outputs at creation time
	AdditionalWriters []io.Writer
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Logger wraps zerolog.Logger with component support
type Logger struct {
	zlog      zerolog.Logger
	component string
	output    *writerSet
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// New creates a new Logger with the specified configuration
func New(config *LoggerConfig) *Logger {
	if config == nil {
		config = &LoggerConfig{
			Level:         LevelInfo,
			ConsoleOutput: true,
		}
	}

	writers := &writerSet{}

	if config.ConsoleOutput {
		writers.add(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "2006-01-02 15:04:05",
			NoColor:    false,
		})
	}

	for _, w := range config.AdditionalWriters {
		writers.add(w)
	}

	zlog := zerolog.New(writers).With().Timestamp().Logger()

	switch config.Level {
	case LevelDebug:
		zlog = zlog.Level(zerolog.DebugLevel)
	case LevelInfo:
		zlog = zlog.Level(zerolog.InfoLevel)
	case LevelWarn:
		zlog = zlog.Level(zerolog.WarnLevel)
	case LevelError:
		zlog = zlog.Level(zerolog.ErrorLevel)
	default:
		zlog = zlog.Level(zerolog.InfoLevel)
	}

	return &Logger{
		zlog:   zlog,
		output: writers,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithComponent returns a child logger that tags output with the given component name
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		zlog:      l.zlog.With().Str("component", component).Logger(),
		component: component,
		output:    l.output,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AddWriter appends a writer to the root output set. Child loggers created with
// WithComponent share the same set, so they receive output from writers added later
func (l *Logger) AddWriter(w io.Writer) {
	if l == nil || l.output == nil || w == nil {
		return
	}

	l.output.add(w)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Debug returns a debug level event
func (l *Logger) Debug() *zerolog.Event {
	return l.zlog.Debug()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Info returns an info level event
func (l *Logger) Info() *zerolog.Event {
	return l.zlog.Info()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Error returns an error level event
func (l *Logger) Error() *zerolog.Event {
	return l.zlog.Error()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Warn returns a warn level event
func (l *Logger) Warn() *zerolog.Event {
	return l.zlog.Warn()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetZerolog returns the underlying zerolog.Logger for advanced usage
func (l *Logger) GetZerolog() zerolog.Logger {
	return l.zlog
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Component returns the current component name
func (l *Logger) Component() string {
	return l.component
}
