package cron

import (
	"net/http"
	"time"

	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/cardcache"
	"github.com/geerew/off-course/utils/logger"
	"github.com/robfig/cron/v3"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CronConfig holds the configuration needed to create a Cron scheduler
type CronConfig struct {
	DbManager *database.DatabaseManager
	AppFs     *appfs.AppFs
	CardCache *cardcache.CardCache

	// Loggers are scoped by the app (component tags) before being passed in
	CourseAvailabilityLogger *logger.Logger
	CardCacheWarmLogger      *logger.Logger
	ReleaseCheckerLogger     *logger.Logger
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Cron manages cron jobs
//
// When created, call c.Start() to start the scheduler
type Cron struct {
	// Services
	CourseAvailability *courseAvailability
	CardCacheWarm      *cardCacheWarm
	ReleaseChecker     *releaseChecker

	// Cron scheduler
	cron *cron.Cron
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewCronScheduler creates a new Cron scheduler
func NewCronScheduler(config *CronConfig) *Cron {
	c := &Cron{cron: cron.New()}

	// Course availability
	c.CourseAvailability = &courseAvailability{
		db:        config.DbManager.DataDb,
		dao:       dao.New(config.DbManager.DataDb),
		appFs:     config.AppFs,
		logger:    config.CourseAvailabilityLogger,
		batchSize: 200,
	}

	c.cron.AddFunc("@every 5m", func() { c.CourseAvailability.run() })

	// Card cache warm
	if config.CardCache != nil {
		c.CardCacheWarm = &cardCacheWarm{
			db:        config.DbManager.DataDb,
			dao:       dao.New(config.DbManager.DataDb),
			cardCache: config.CardCache,
			logger:    config.CardCacheWarmLogger,
		}
	}

	// Release checker
	c.ReleaseChecker = &releaseChecker{
		logger:     config.ReleaseCheckerLogger,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	c.cron.AddFunc("@every 5m", func() { c.ReleaseChecker.run() })

	return c
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Start starts the cron scheduler
func (c *Cron) Start() {
	// Run startup jobs immediately (not on the recurring schedule above)
	go func() { c.CourseAvailability.run() }()
	if c.CardCacheWarm != nil {
		go func() {
			if err := c.CardCacheWarm.run(); err != nil {
				c.CardCacheWarm.logger.Error().Err(err).Msg("Failed to warm card serve index")
			}
		}()
	}
	go func() { c.ReleaseChecker.run() }()

	c.cron.Start()
}
