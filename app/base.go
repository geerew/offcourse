package app

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/cron"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/auth"
	"github.com/geerew/off-course/utils/cardcache"
	"github.com/geerew/off-course/utils/coursemetadata"
	"github.com/geerew/off-course/utils/coursescan"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/media"
	"github.com/geerew/off-course/utils/media/hls"
	"github.com/geerew/off-course/utils/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// App represents the application with dependencies
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
	Cron           *cron.Cron

	// Configuration
	Config *Config

	// Internal
	dbLogger     *logger.DbWriter
	bootstrapped atomic.Int32
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Mode represents the application mode
type AppMode int

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// AppMode constants
//  1. Production
//  2. Development
//  3. Test
const (
	AppModeProd AppMode = iota
	AppModeDev
	AppModeTest
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var (
	// ffmpegOnce caches the FFmpeg lookup
	ffmpegOnce sync.Once
	// cachedFFmpeg is the cached FFmpeg lookup result
	cachedFFmpeg *media.FFmpeg
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Config holds application configuration
type Config struct {
	HttpAddr     string
	DataDir      string
	EnableSignup bool
	Debug        bool
	AppMode      AppMode
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// New creates a new App instance with all dependencies initialized
func NewApp(ctx context.Context, config *Config) (*App, error) {
	logLevel := logger.LevelInfo
	if config.Debug {
		logLevel = logger.LevelDebug
	}

	// AppFS (filesystem)
	var appFs *appfs.AppFs
	if config.AppMode == AppModeTest {
		appFs = appfs.New(afero.NewMemMapFs())
	} else {
		appFs = appfs.New(afero.NewOsFs())
	}

	// FFmpeg
	var ffmpeg *media.FFmpeg
	if config.AppMode == AppModeTest {
		ffmpeg = getCachedFFmpegOrPanic()
	} else {
		var err error
		ffmpeg, err = media.NewFFmpeg()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize FFmpeg: %w", err)
		}
	}

	// Database manager
	dbManagerConfig := &database.DatabaseManagerConfig{
		DataDir: config.DataDir,
		AppFs:   appFs,
		Testing: config.AppMode == AppModeTest,
	}

	dbManager, err := database.NewSQLiteManager(dbManagerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database manager: %w", err)
	}

	// Create logger
	var appLogger *logger.Logger
	var dbLogger *logger.DbWriter

	if config.AppMode == AppModeTest {
		appLogger = logger.NilLogger()
	} else {
		logDao := dao.New(dbManager.LogsDb)

		// Create a db logger
		dbLoggerConfig := &logger.DbWriterConfig{
			BatchSize:     100,
			FlushInterval: 5 * time.Second,
		}

		dbLogger = logger.NewDbWriter(logDao.CreateLogsBatch, dbLoggerConfig)

		// Create the app logger
		appLogger = logger.New(&logger.Config{
			Level:         logLevel,
			ConsoleOutput: true,
			DbWriter:      dbLogger,
		})

		if appLogger == nil {
			dbLogger.Close()
			return nil, fmt.Errorf("failed to initialize logger")
		}
	}

	// HLS Transcoder
	transcoderConfig := &hls.TranscoderConfig{
		CachePath: config.DataDir,
		HwAccel:   hls.DetectHardwareAccel(appLogger.WithHLS()),
		AppFs:     appFs,
		Logger:    appLogger.WithHLS(),
		Dao:       dao.New(dbManager.DataDb),
	}

	transcoder, err := hls.NewTranscoder(transcoderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create HLS transcoder: %w", err)
	}

	// Card Cache
	cardCacheConfig := &cardcache.CardCacheConfig{
		CachePath: config.DataDir,
		AppFs:     appFs,
		Logger:    appLogger.WithCardCache(),
		FFmpeg:    ffmpeg,
	}

	cardCache, err := cardcache.New(cardCacheConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create card cache: %w", err)
	}

	// Course scanner
	courseScan := coursescan.New(&coursescan.CourseScanConfig{
		Db:        dbManager.DataDb,
		AppFs:     appFs,
		Logger:    appLogger.WithCourseScan(),
		FFmpeg:    ffmpeg,
		CardCache: cardCache,
	})

	// Metadata writer (for oc.json files)
	metadataWriter := coursemetadata.NewMetadataWriter(appFs.Fs, appLogger.WithCourseMetadata())

	// Ensure fallback card exists
	fallbackPath := cardCache.GetFallbackPath()
	if err := cardCache.EnsureFallbackCard(fallbackPath); err != nil {
		return nil, fmt.Errorf("failed to ensure fallback card exists: %w", err)
	}

	// Cron scheduler
	cronScheduler := cron.NewCronScheduler(&cron.CronConfig{
		DbManager: dbManager,
		AppFs:     appFs,
		Logger:    appLogger,
	})

	app := &App{
		Logger:         appLogger,
		AppFs:          appFs,
		FFmpeg:         ffmpeg,
		DbManager:      dbManager,
		Config:         config,
		Transcoder:     transcoder,
		CardCache:      cardCache,
		CourseScan:     courseScan,
		MetadataWriter: metadataWriter,
		dbLogger:       dbLogger,
		Cron:           cronScheduler,
	}

	// Bootstrap
	if err := app.bootstrap(); err != nil {
		return nil, fmt.Errorf("failed to bootstrap: %w", err)
	}

	return app, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewTestApp creates an app instance that is suitable for running testcases
func NewTestApp(t *testing.T) *App {
	t.Helper()

	app, err := NewApp(context.Background(), &Config{
		HttpAddr:     "127.0.0.1:9081",
		DataDir:      "./oc_data",
		AppMode:      AppModeTest,
		EnableSignup: true,
		Debug:        false,
	})

	require.NoError(t, err)
	require.NotNil(t, app)

	return app
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Close closes all resources that need cleanup
func (a *App) Close() error {
	if a.dbLogger != nil {
		return a.dbLogger.Close()
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsBootstrapped checks if the application has been bootstrapped
func (a *App) IsBootstrapped() bool {
	return a.bootstrapped.Load() == 1
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SetBootstrapped sets the application as bootstrapped
func (a *App) SetBootstrapped() {
	a.bootstrapped.Store(1)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UnsetBootstrapped sets the application as not bootstrapped
func (a *App) UnsetBootstrapped() {
	a.bootstrapped.Store(0)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// bootstrap checks if the app is bootstrapped and generates a bootstrap token if not,
// enabled the user to create the first admin user
func (a *App) bootstrap() error {
	appDao := dao.New(a.DbManager.DataDb)
	count, err := appDao.CountUsers(
		context.Background(),
		dao.NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ROLE: types.UserRoleAdmin}),
	)

	if err != nil {
		return fmt.Errorf("failed to count admin users: %w", err)
	}

	if count == 0 {
		a.bootstrapped.Store(0)

		bootstrapToken, err := auth.GenerateBootstrapToken(a.Config.DataDir, a.AppFs.Fs)
		if err != nil {
			return fmt.Errorf("failed to generate bootstrap token: %w", err)
		}

		bootstrapURL := fmt.Sprintf("http://%s/auth/bootstrap/%s", a.Config.HttpAddr, bootstrapToken.Token)
		a.Logger.WithApp().Info().
			Str("bootstrap_url", bootstrapURL).
			Str("expires_in", "5 minutes").
			Msg("Bootstrap required")
	} else {
		a.bootstrapped.Store(1)

		// Clean up any existing bootstrap tokens
		if err := auth.DeleteBootstrapToken(a.Config.DataDir, a.AppFs.Fs); err != nil {
			a.Logger.WithApp().Warn().Err(err).Msg("Failed to delete bootstrap token")
		}

		a.Logger.WithApp().Info().Msg("Application bootstrapped")
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getCachedFFmpegOrPanic will look call NewFFmpeg once then cache the result. This is
// useful for test cases where we don't want to look up the FFmpeg executable
// repeatedly
//
// Panics when NewFFmpeg errors
func getCachedFFmpegOrPanic() *media.FFmpeg {
	ffmpegOnce.Do(func() {
		ff, err := media.NewFFmpeg()
		if err != nil {
			panic(err)
		}

		cachedFFmpeg = ff
	})

	return cachedFFmpeg
}
