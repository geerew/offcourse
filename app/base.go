package app

import (
	"context"
	"time"

	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/cardcache"
	"github.com/geerew/off-course/utils/coursemetadata"
	"github.com/geerew/off-course/utils/coursescan"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/media"
	"github.com/geerew/off-course/utils/media/hls"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// App represents the application and contains all dependencies
type App struct {
	// Core dependencies
	Logger    *logger.Logger
	AppFs     *appfs.AppFs
	FFmpeg    *media.FFmpeg
	DbManager *database.DatabaseManager

	// Services
	CourseScan     *coursescan.CourseScan
	Transcoder     *hls.Transcoder
	CardCache      *cardcache.CardCache
	MetadataWriter *coursemetadata.MetadataWriter

	// Configuration
	Config *Config

	// Internal
	dbWriter *logger.DbWriter
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Config holds application configuration
type Config struct {
	HttpAddr     string
	DataDir      string
	IsDev        bool
	EnableSignup bool
	IsDebug      bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// New creates a new App instance with all dependencies initialized
func New(ctx context.Context, config *Config) (*App, error) {
	// Determine log level
	logLevel := logger.LevelInfo
	if config.IsDebug {
		logLevel = logger.LevelDebug
	}

	// AppFS (filesystem)
	appFs := appfs.New(afero.NewOsFs())

	// FFmpeg
	ffmpeg, err := media.NewFFmpeg()
	if err != nil {
		return nil, &InitializationError{Message: "Failed to initialize FFmpeg", Err: err}
	}

	// Database manager
	dbManager, err := database.NewSQLiteManager(&database.DatabaseManagerConfig{
		DataDir: config.DataDir,
		AppFs:   appFs,
	})

	if err != nil {
		return nil, &InitializationError{Message: "Failed to create database manager", Err: err}
	}

	// Create DAO for database logging
	logDao := dao.New(dbManager.LogsDb)

	// Create database log writer with batching
	dbWriter := logger.CreateDbWriter(logDao, &logger.DbWriterConfig{
		BatchSize:     100,
		FlushInterval: 5 * time.Second,
	})

	// Create logger with database writer
	appLogger := logger.New(&logger.Config{
		Level:         logLevel,
		ConsoleOutput: true,
		DbWriter:      dbWriter,
	})

	if appLogger == nil {
		dbWriter.Close()
		return nil, &InitializationError{Message: "Failed to initialize logger"}
	}

	// Create app instance first (needed for service initialization)
	app := &App{
		Logger:    appLogger,
		AppFs:     appFs,
		FFmpeg:    ffmpeg,
		DbManager: dbManager,
		Config:    config,
		dbWriter:  dbWriter,
	}

	// HLS Transcoder
	transcoder, err := hls.NewTranscoder(&hls.TranscoderConfig{
		CachePath: app.Config.DataDir,
		HwAccel:   hls.DetectHardwareAccel(app.Logger.WithHLS()),
		AppFs:     app.AppFs,
		Logger:    app.Logger.WithHLS(),
		Dao:       dao.New(app.DbManager.DataDb),
	})

	if err != nil {
		return nil, &InitializationError{Message: "Failed to create HLS transcoder", Err: err}
	}

	app.Transcoder = transcoder

	// Card Cache
	cardCache, err := cardcache.NewCardCache(&cardcache.CardCacheConfig{
		CachePath: app.Config.DataDir,
		AppFs:     app.AppFs,
		Logger:    app.Logger.WithCardCache(),
		FFmpeg:    app.FFmpeg,
	})

	if err != nil {
		return nil, &InitializationError{Message: "Failed to create card cache", Err: err}
	}

	app.CardCache = cardCache

	// Course scanner
	app.CourseScan = coursescan.New(&coursescan.CourseScanConfig{
		Db:        app.DbManager.DataDb,
		AppFs:     app.AppFs,
		Logger:    app.Logger.WithCourseScan(),
		FFmpeg:    app.FFmpeg,
		CardCache: cardCache,
	})

	// Metadata writer for oc.json files
	app.MetadataWriter = coursemetadata.NewMetadataWriter(app.AppFs.Fs, app.Logger.WithCourseMetadata())

	// Ensure fallback card exists
	fallbackPath := cardCache.GetFallbackPath()
	if err := cardCache.EnsureFallbackCard(fallbackPath); err != nil {
		return nil, &InitializationError{
			Message: "Failed to ensure fallback card exists",
			Err:     err,
		}
	}

	return app, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Close closes all resources that need cleanup (e.g., database log writer)
func (a *App) Close() error {
	if a.dbWriter != nil {
		return a.dbWriter.Close()
	}
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// InitializationError represents an error during app initialization
type InitializationError struct {
	Message string
	Err     error
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Error returns the error message
func (e *InitializationError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Unwrap returns the wrapped error
func (e *InitializationError) Unwrap() error {
	return e.Err
}
