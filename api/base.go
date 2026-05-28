package api

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/geerew/off-course/app"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/session"
	"github.com/gofiber/fiber/v2"
	fibersession "github.com/gofiber/fiber/v2/middleware/session"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MiddlewareFactory defines a function that creates middleware with access to the router
type MiddlewareFactory func(r *Router) fiber.Handler

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Router defines a router
type Router struct {
	fiberApp       *fiber.App
	app            *app.App
	appDao         *dao.DAO
	logDao         *dao.DAO
	sessionManager *session.SessionManager
	logger         *logger.Logger
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewRouter creates a new router from an App instance
func NewRouter(application *app.App) *Router {
	r := &Router{
		app:    application,
		appDao: dao.New(application.DbManager.DataDb),
		logDao: dao.New(application.DbManager.LogsDb),
		logger: application.Logger.WithComponent(string(app.ComponentAPI)),
	}

	r.createSessionStore()

	r.fiberApp = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	r.initMiddleware()
	r.initRoutes()

	return r
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Serve serves the API and UI
func (r *Router) Serve() error {
	ln, err := net.Listen("tcp", r.app.Config.HttpAddr)
	if err != nil {
		return err
	}

	r.logger.Info().
		Str("url", fmt.Sprintf("http://%s", r.app.Config.HttpAddr)).
		Msg("Server started")

	return r.fiberApp.Listener(ln)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Private
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initMiddleware initializes the middleware
func (r *Router) initMiddleware() {
	// Middleware
	r.fiberApp.Use(requestLoggingMiddleware(r.logger))
	r.fiberApp.Use(corsMiddleWare())
	r.fiberApp.Use(bootstrapMiddleware(r))
	r.fiberApp.Use(authMiddleware(r))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initRoutes initializes the routes
func (r *Router) initRoutes() {
	// UI
	r.bindUi()

	// API routes
	r.initAuthRoutes()
	r.initFsRoutes()
	r.initCourseRoutes()
	r.initScanRoutes()
	r.initTagRoutes()
	r.initUserRoutes()
	r.initLogRoutes()
	r.initRecoveryRoutes()
	r.initHlsRoutes()
	r.initVersionRoutes()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// createSessionStore creates the session store
func (r *Router) createSessionStore() {
	config := fibersession.Config{
		KeyLookup:      "cookie:session",
		Expiration:     7 * (24 * time.Hour),
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
	}

	sqliteStorage := session.NewSqliteStorage(r.app.DbManager.DataDb, 10*time.Second)

	r.sessionManager = session.New(r.app.DbManager.DataDb, config, sqliteStorage)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// apiGroup returns a new API router group
func (r *Router) apiGroup(groupPath string) fiber.Router {
	return r.fiberApp.Group("/api/" + groupPath)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SetTestMiddleware replaces the middleware stack with test middleware
//
// FOR TESTING PURPOSES ONLY
func (r *Router) SetTestMiddleware(factories ...MiddlewareFactory) {
	// Clear existing middleware by creating a new Fiber app with same config
	r.fiberApp = fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Default error handler that returns JSON
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"message": err.Error(),
			})
		},
	})

	// Apply test middleware
	for _, factory := range factories {
		r.fiberApp.Use(factory(r))
	}

	// Re-initialize routes (they depend on the Fiber app)
	r.initRoutes()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test is a test helper that wraps FiberApp.Test() for testing purposes
func (r *Router) Test(req *http.Request, msTimeout ...int) (*http.Response, error) {
	return r.fiberApp.Test(req, msTimeout...)
}
