package utils

import "errors"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var (
	// Generic
	ErrNilPtr        = errors.New("nil pointer")
	ErrNotPtr        = errors.New("requires a pointer")
	ErrNotModeler    = errors.New("does not implement Modeler interface")
	ErrEmbedded      = errors.New("embedded struct does not implement Definer interface")
	ErrInvalidValue  = errors.New("invalid value")
	ErrInvalidColumn = errors.New("invalid column")
	ErrInvalidPluck  = errors.New("pluck is only valid when selecting a single column")
	ErrNotStruct     = errors.New("not a struct")
	ErrNotSlice      = errors.New("not a slice")
	ErrNoTable       = errors.New("table name cannot be empty")

	// API
	ErrApiQueryParse = errors.New("list query parse error")

	// DB
	ErrWhere     = errors.New("where clause cannot be empty")
	ErrPrincipal = errors.New("principal not found in context")

	// Model
	ErrId                  = errors.New("id cannot be empty")
	ErrCourseId            = errors.New("course id cannot be empty")
	ErrCourseNotFound      = errors.New("course not found")
	ErrLessonId            = errors.New("lesson id cannot be empty")
	ErrKey                 = errors.New("key cannot be empty")
	ErrUsername            = errors.New("username cannot be empty")
	ErrUserPassword        = errors.New("user password cannot be empty")
	ErrLogMessage          = errors.New("log message cannot be empty")
	ErrUserId              = errors.New("user id cannot be empty")
	ErrAssetId             = errors.New("asset id cannot be empty")
	ErrTag                 = errors.New("tag cannot be empty")
	ErrTitle               = errors.New("title cannot be empty")
	ErrPrefix              = errors.New("prefix cannot be empty or less than zero")
	ErrPath                = errors.New("path cannot be empty")
	ErrAssetCourseRelation = errors.New("asset does not belong to course")

	// Media
	ErrInvalidFFProbePath = errors.New("ffprobe path is invalid")
	ErrFFProbeNotFound    = errors.New("ffprobe not found in path")
	ErrFFProbeUnavailable = errors.New("ffprobe unavailable")
	ErrFFProbePathEmpty   = errors.New("ffprobe path cannot be empty")
	ErrInvalidFFmpegPath  = errors.New("ffmpeg path is invalid")
	ErrFFmpegNotFound     = errors.New("ffmpeg not found in path")
	ErrFFmpegUnavailable  = errors.New("ffmpeg unavailable")
	ErrFFmpegPathEmpty    = errors.New("ffmpeg path cannot be empty")
)
