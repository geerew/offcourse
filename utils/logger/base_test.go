package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestNew(t *testing.T) {
	// Test default config when nil is passed
	t.Run("nil config", func(t *testing.T) {
		l := New(nil)
		require.NotNil(t, l)
	})

	// Test discarding output when no writers are configured
	t.Run("no writers", func(t *testing.T) {
		l := New(&LoggerConfig{
			Level:         LevelInfo,
			ConsoleOutput: false,
		})
		require.NotNil(t, l)
		l.Info().Msg("discarded")
	})

	// Test additional writer receives JSON output
	t.Run("additional writer", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(&LoggerConfig{
			Level:             LevelInfo,
			ConsoleOutput:     false,
			AdditionalWriters: []io.Writer{&buf},
		})

		l.Info().Str("key", "value").Msg("hello")

		var line map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &line))
		require.Equal(t, "hello", line["message"])
		require.Equal(t, "value", line["key"])
	})

	// Test minimum log levels are applied
	t.Run("levels", func(t *testing.T) {
		tests := []struct {
			level    LogLevel
			expected zerolog.Level
		}{
			{LevelDebug, zerolog.DebugLevel},
			{LevelInfo, zerolog.InfoLevel},
			{LevelWarn, zerolog.WarnLevel},
			{LevelError, zerolog.ErrorLevel},
		}

		for _, tt := range tests {
			l := New(&LoggerConfig{Level: tt.level, ConsoleOutput: false})
			require.Equal(t, tt.expected, l.GetZerolog().GetLevel())
		}
	})

	// Test unknown level defaults to info
	t.Run("unknown level", func(t *testing.T) {
		l := New(&LoggerConfig{Level: LogLevel(99), ConsoleOutput: false})
		require.Equal(t, zerolog.InfoLevel, l.GetZerolog().GetLevel())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogger_WithComponent(t *testing.T) {
	// Test component field is included in JSON output
	t.Run("json component", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(&LoggerConfig{
			Level:             LevelInfo,
			ConsoleOutput:     false,
			AdditionalWriters: []io.Writer{&buf},
		})

		child := l.WithComponent("api")
		require.Equal(t, "api", child.Component())
		child.Info().Msg("api call")

		var line map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &line))
		require.Equal(t, "api", line["component"])
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestLogger_AddWriter(t *testing.T) {
	// Test a writer added after New receives later log lines, including from WithComponent
	t.Run("late writer", func(t *testing.T) {
		var early, late bytes.Buffer
		l := New(&LoggerConfig{
			Level:             LevelInfo,
			ConsoleOutput:     false,
			AdditionalWriters: []io.Writer{&early},
		})

		child := l.WithComponent("api")
		child.Info().Msg("before")

		l.AddWriter(&late)
		child.Info().Msg("after")

		require.Contains(t, early.String(), "before")
		require.NotContains(t, late.String(), "before")
		require.Contains(t, late.String(), "after")
	})

	// Test AddWriter is a no-op on NilLogger
	t.Run("nil logger", func(t *testing.T) {
		var buf bytes.Buffer
		l := NilLogger()
		l.AddWriter(&buf)
		l.Info().Msg("silent")
		require.Empty(t, buf.String())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test NilLogger discards output without panicking
func TestNilLogger(t *testing.T) {
	l := NilLogger()
	require.NotNil(t, l)
	l.Info().Msg("silent")
	l.Error().Msg("also silent")
}
