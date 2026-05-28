package app

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Component names a log source for filtering and structured output. Add a constant here when
// wiring a new subsystem logger in NewApp
type Component string

const (
	ComponentApp            Component = "app"
	ComponentAPI            Component = "api"
	ComponentHLS            Component = "hls"
	ComponentCourseScan     Component = "coursescan"
	ComponentCardCache      Component = "cardcache"
	ComponentCourseMetadata Component = "coursemetadata"
	ComponentCron           Component = "cron"
)
