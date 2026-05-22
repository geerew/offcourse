package api

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/filesystem"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
	"github.com/gofiber/fiber/v2"
	fiberfs "github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// errorResponse is a helper method to return an error response
func errorResponse(c *fiber.Ctx, status int, message string, err error) error {
	resp := fiber.Map{
		"message": message,
	}

	if err != nil {
		resp["error"] = err.Error()
	}

	// Store error details for centralized logging in middleware
	if status >= 400 {
		c.Locals("api_error_message", message)
		if err != nil {
			c.Locals("api_error_detail", err.Error())
		}
	}

	return c.Status(status).JSON(resp)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// validatePassword validates a password
func validatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	if len(password) > 128 {
		return fmt.Errorf("password must be no more than 128 characters")
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const bufferSize = 1024 * 8                 // 8KB per chunk, adjust as needed
const maxInitialChunkSize = 1024 * 1024 * 5 // 5MB, adjust as needed

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// principalCtx is a helper method to get the principal and build a new context
func principalCtx(c *fiber.Ctx) (types.Principal, context.Context, error) {
	principal, ok := c.Locals(types.PrincipalContextKey).(types.Principal)
	if !ok {
		return types.Principal{}, nil, fmt.Errorf("missing principal")
	}

	ctx := c.UserContext()
	ctx = context.WithValue(ctx, types.PrincipalContextKey, principal)

	return principal, ctx, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// paginationFromCtx builds pagination from page and perPage query parameters.
func paginationFromCtx(c *fiber.Ctx) *pagination.Pagination {
	return pagination.New(
		pagination.ParsePage(c.Query(pagination.PageQueryParam)),
		pagination.ParsePerPage(c.Query(pagination.PerPageQueryParam)),
	)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// handleVideo handles the video streaming logic
func handleVideo(c *fiber.Ctx, fs *filesystem.FS, asset *models.Asset) error {
	// Open the video
	file, err := fs.Open(asset.Path)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error opening file", err)
	}
	defer file.Close()

	// Get the file info
	fileInfo, err := file.Stat()
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error getting file info", err)
	}

	// Get the range header and return the entire video if there is no range header
	rangeHeader := c.Get("Range", "")
	if rangeHeader == "" {
		return fiberfs.SendFile(c, afero.NewHttpFs(fs), asset.Path)
	}

	// Parse the "bytes=START-END" format
	bytesPos := strings.Split(rangeHeader, "=")[1]
	rangeStartEnd := strings.Split(bytesPos, "-")
	start, _ := strconv.Atoi(rangeStartEnd[0])
	var end int

	if len(rangeStartEnd) == 2 && rangeStartEnd[0] == "0" && rangeStartEnd[1] == "1" {
		start = 0
		end = 1
	} else {
		end = start + maxInitialChunkSize - 1
		if end >= int(fileInfo.Size()) {
			end = int(fileInfo.Size()) - 1
		}
	}

	if start > end {
		return errorResponse(c, fiber.StatusBadRequest, "Range start cannot be greater than end", fmt.Errorf("range start is greater than end"))
	}

	// Setting required response headers
	c.Set(fiber.HeaderContentRange, fmt.Sprintf("bytes %d-%d/%d", start, end, fileInfo.Size()))
	c.Set(fiber.HeaderContentLength, strconv.Itoa(end-start+1))
	c.Set(fiber.HeaderContentType, mime.TypeByExtension(filepath.Ext(asset.Path)))
	c.Set(fiber.HeaderAcceptRanges, "bytes")

	// Set the status code to 206 Partial Content
	c.Status(fiber.StatusPartialContent)

	file.Seek(int64(start), 0)
	buffer := make([]byte, bufferSize)
	bytesToSend := end - start + 1

	// Respond in chunks
	for bytesToSend > 0 {
		bytesRead, err := file.Read(buffer)
		if err != nil {
			break
		}

		if bytesRead > bytesToSend {
			bytesRead = bytesToSend
		}

		c.Write(buffer[:bytesRead])
		bytesToSend -= bytesRead
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// handleText handles serving text files and markdown files
func handleText(c *fiber.Ctx, fs *filesystem.FS, asset *models.Asset) error {
	file, err := fs.Open(asset.Path)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error opening text file", err)
	}
	defer file.Close()

	raw, err := afero.ReadAll(file)
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error reading text file", err)
	}

	c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
	return c.Status(fiber.StatusOK).Send(raw)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// protectedRoute protects a route
//
// Example:
//
//	group.Get("/my-route", protectedRoute, myHandler)
func protectedRoute(c *fiber.Ctx) error {
	principal, _, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	if principal.Role != types.UserRoleAdmin {
		return errorResponse(c, fiber.StatusForbidden, "User is not an admin", nil)
	}

	return c.Next()
}
