package cron

import (
	"net/http"
	"time"

	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/logger"
	"github.com/robfig/cron/v3"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CronConfig holds the configuration needed to create a Cron scheduler
type CronConfig struct {
	DbManager *database.DatabaseManager
	AppFs     *appfs.AppFs
	Logger    *logger.Logger
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Cron manages cron jobs
//
// When created, call c.Start() to start the scheduler
type Cron struct {
	// Services
	CourseAvailability *courseAvailability
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
		logger:    config.Logger.WithCron(),
		batchSize: 200,
	}

	c.cron.AddFunc("@every 5m", func() { c.CourseAvailability.run() })

	// Release checker
	c.ReleaseChecker = &releaseChecker{
		logger:     config.Logger.WithCron(),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}

	c.cron.AddFunc("@every 5m", func() { c.ReleaseChecker.run() })

	return c
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Start starts the cron scheduler
func (c *Cron) Start() {
	// Run some jobs immediately
	go func() { c.CourseAvailability.run() }()
	go func() { c.ReleaseChecker.run() }()

	c.cron.Start()
}
