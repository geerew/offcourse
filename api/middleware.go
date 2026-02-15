package api

// TODO When 2 windows are open with bootstrap, and the first one bootstraps, the second should
//   1) error on submit or
//   2) redirect to the login page on refresh

// TODO Tidy the middleware code, it is a bit messy

import (
	"strings"
	"time"

	"github.com/geerew/off-course/app"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/types"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// corsMiddleWare creates a CORS middleware
func corsMiddleWare() fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET, POST, PUT, DELETE, HEAD, PATCH",
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// requestLoggingMiddleware creates a request logging middleware
func requestLoggingMiddleware(logger *logger.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start)

		// Only log API requests (paths starting with /api)
		path := c.Path()
		if !strings.HasPrefix(path, "/api/") {
			return err
		}

		status := c.Response().StatusCode()
		apiLogger := logger.WithAPI()

		// Pull any error info stored by errorResponse
		errMsg, _ := c.Locals("api_error_message").(string)
		errDetail, _ := c.Locals("api_error_detail").(string)

		switch {
		case status >= 500:
			evt := apiLogger.Error().
				Str("method", c.Method()).
				Int("status", status).
				Dur("duration", duration).
				Str("ip", c.IP())
			if errMsg != "" {
				evt = evt.Str("error_message", errMsg)
			}
			if errDetail != "" {
				evt = evt.Str("error_detail", errDetail)
			}
			evt.Msg(c.Path())
		case status >= 400:
			evt := apiLogger.Warn().
				Str("method", c.Method()).
				Int("status", status).
				Dur("duration", duration).
				Str("ip", c.IP())
			if errMsg != "" {
				evt = evt.Str("error_message", errMsg)
			}
			if errDetail != "" {
				evt = evt.Str("error_detail", errDetail)
			}
			evt.Msg(c.Path())
		default:
			apiLogger.Debug().
				Str("method", c.Method()).
				Int("status", status).
				Dur("duration", duration).
				Str("ip", c.IP()).
				Msg(c.Path())
		}

		return err
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// bootstrapMiddleware checks if the app is bootstrapped. If not, it redirects
// to /auth/bootstrap
//
// Bootstrapping is the process of setting up the app for the first time. It involves
// the creation of 1 admin user, which the /auth/bootstrap endpoint handles
func bootstrapMiddleware(r *Router) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// If not bootstrapped, force everything through /auth/bootstrap or
		// /api/auth/bootstrap
		if !r.app.IsBootstrapped() {
			if r.isDevUIPath(path) || r.isProdUIPath(path) || r.isStaticPath(path) {
				return c.Next()
			}

			// API check
			if strings.HasPrefix(path, "/api/") {
				if strings.HasPrefix(path, "/api/auth/bootstrap/") {
					c.Locals("bootstrapping", true)
					return c.Next()
				} else {
					return errorResponse(c, fiber.StatusForbidden, "app is not bootstrapped", nil)
				}
			}

			// UI check
			if strings.HasPrefix(path, "/auth/bootstrap") {
				c.Locals("bootstrapping", true)
				return c.Next()
			} else {
				return c.Redirect("/auth/bootstrap")
			}
		}

		// If bootstrapped and someone accesses bootstrap URL, redirect appropriately
		if strings.HasPrefix(path, "/auth/bootstrap/") {
			// Check if user is logged in
			session, err := r.sessionManager.Get(c)

			if err != nil || session.Fresh() {
				// Error getting session or session is fresh (no valid cookie), redirect to login
				return c.Redirect("/auth/login")
			}

			// Logged in (session exists and is not fresh), redirect to home
			return c.Redirect("/")
		}

		return c.Next()
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// authMiddleware authenticates the request
func authMiddleware(r *Router) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		if bootstrapping, _ := c.Locals("bootstrapping").(bool); bootstrapping {
			return c.Next()
		}

		isLogout := strings.HasPrefix(path, "/api/auth/logout")
		isAuthUI := strings.HasPrefix(path, "/auth/")

		if r.isDevUIPath(path) || r.isProdUIPath(path) || r.isStaticPath(path) || isLogout {
			return c.Next()
		}

		session, err := r.sessionManager.Get(c)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		sessionFresh := session.Fresh()

		if sessionFresh {
			// Is API request
			if strings.HasPrefix(path, "/api/") {
				if strings.HasPrefix(path, "/api/auth/login") || (r.app.Config.EnableSignup && strings.HasPrefix(path, "/api/auth/register")) ||
					strings.HasPrefix(path, "/api/auth/signup-status") || strings.HasPrefix(path, "/api/admin/recovery") {
					return c.Next()
				}
				return c.SendStatus(fiber.StatusForbidden)
			}

			if isAuthUI {
				if !r.app.Config.EnableSignup && strings.HasPrefix(path, "/auth/register") {
					return c.Redirect("/auth/login")
				}

				return c.Next()
			}

			return c.Redirect("/auth/login")
		}

		if isAuthUI {
			return c.Redirect("/")
		}

		// Allow public auth endpoints to proceed even with a session cookie
		if strings.HasPrefix(path, "/api/auth/") {
			if strings.HasPrefix(path, "/api/auth/login") ||
				(r.app.Config.EnableSignup && strings.HasPrefix(path, "/api/auth/register")) ||
				strings.HasPrefix(path, "/api/auth/signup-status") ||
				strings.HasPrefix(path, "/api/auth/bootstrap/") {

				return c.Next()
			}
		}

		// Validate session for protected endpoints
		userID, ok1 := session.Get("id").(string)
		userRole, ok2 := session.Get("role").(string)
		if !ok1 || !ok2 || userID == "" || userRole == "" {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		if userRole != "admin" && r.isProtectedUIPage(path) {
			return c.Redirect("/")
		}

		role := types.UserRole(userRole)
		principal := types.Principal{
			UserID: userID,
			Role:   role,
		}

		// for your UI/router logic:
		c.Locals(types.PrincipalContextKey, principal)

		return c.Next()
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isProdUIPath checks if the request is for a sveltekit asset when running in production mode
func (r *Router) isProdUIPath(path string) bool {
	if r.app.Config.AppMode != app.AppModeDev && strings.HasPrefix(path, "/_app/") {
		return true
	}

	return false
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isDevUIPath checks if the request is for a sveltekit path when running in dev mode
func (r *Router) isDevUIPath(path string) bool {
	if r.app.Config.AppMode == app.AppModeDev &&
		(strings.HasPrefix(path, "/node_modules/") ||
			strings.HasPrefix(path, "/.svelte-kit/") ||
			strings.HasPrefix(path, "/src/") ||
			strings.HasPrefix(path, "/@")) {
		return true
	}

	return false
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isStaticPath checks if the request is for a static asset
func (r *Router) isStaticPath(path string) bool {
	if strings.HasPrefix(path, "/apple-touch-icon.png") ||
		strings.HasPrefix(path, "/favicon.") ||
		strings.HasPrefix(path, "/fonts/") ||
		strings.HasPrefix(path, "/web-app-manifest-") {
		return true
	}

	return false
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isProtectedUIPage checks if the request is intended for a protected UI page
func (r *Router) isProtectedUIPage(path string) bool {
	return strings.HasPrefix(path, "/admin")
}
