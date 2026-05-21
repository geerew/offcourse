package logger

import (
	"encoding/json"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// LogBatchWriter batches parsed zerolog lines and flushes models.Log values via onFlush
type LogBatchWriter struct {
	*BatchWriter[*models.Log]
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewLogBatchWriter creates a log batch writer (e.g. onFlush: logDao.CreateLogsBatch)
func NewLogBatchWriter(onFlush BatchFlushFunc[*models.Log], config *BatchWriterConfig) *LogBatchWriter {
	return &LogBatchWriter{
		BatchWriter: NewBatchWriter(onFlush, config),
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Write implements io.Writer by parsing zerolog JSON and appending to the batch
func (w *LogBatchWriter) Write(p []byte) (n int, err error) {
	w.Append(parseZerologJSON(p))
	return len(p), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseZerologJSON converts a zerolog JSON line into a models.Log
func parseZerologJSON(p []byte) *models.Log {
	var raw map[string]interface{}
	if err := json.Unmarshal(p, &raw); err != nil {
		return &models.Log{
			Level:   "info",
			Message: string(p),
		}
	}

	levelStr, _ := raw["level"].(string)
	message, _ := raw["message"].(string)
	component, _ := raw["component"].(string)

	delete(raw, "level")
	delete(raw, "time")
	delete(raw, "message")
	delete(raw, "component")

	var data types.JsonMap
	if len(raw) > 0 {
		data = types.JsonMap(raw)
	}

	log := &models.Log{
		Level:   normalizeLogLevel(levelStr),
		Message: message,
		Data:    data,
	}

	if component != "" {
		if log.Data == nil {
			log.Data = make(types.JsonMap)
		}
		log.Data["component"] = component
	}

	return log
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// normalizeLogLevel returns a known zerolog level string or info for unknown values
func normalizeLogLevel(level string) string {
	switch level {
	case "debug", "info", "warn", "error":
		return level
	default:
		return "info"
	}
}
