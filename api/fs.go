package api

import (
	"path/filepath"

	"github.com/geerew/off-course/utils"
	"github.com/gofiber/fiber/v2"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type fsAPI struct {
	r *Router
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// initFsRoutes initializes the filesystem routes
func (r *Router) initFsRoutes() {
	fsAPI := fsAPI{
		r: r,
	}

	g := r.apiGroup("filesystem")

	g.Get("", protectedRoute, fsAPI.fileSystem)
	g.Get("/:path", protectedRoute, fsAPI.path)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// fileSystem queries the underlying system for available drives
//
// Note: On WSL, the drives will consist of / and /mnt* (ignoring /mnt/wsl*)
func (api fsAPI) fileSystem(c *fiber.Ctx) error {
	// Verify authentication first
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	drives, err := api.r.app.FS.AvailableDrives()
	if err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error looking up available drives", nil)
	}

	directories := make([]*fileInfoResponse, 0)

	normalizedPaths := make([]string, len(drives))
	for _, d := range drives {
		normalizedPath := utils.NormalizeWindowsDrive(d)
		directories = append(directories, &fileInfoResponse{Title: d, Path: normalizedPath})
		normalizedPaths = append(normalizedPaths, normalizedPath)
	}

	// Include path classification; ancestor, course, descendant, none
	if classificationResult, err := api.r.appDao.ClassifyCoursePaths(ctx, normalizedPaths); err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error classifying paths", nil)
	} else {
		for _, dir := range directories {
			dir.Classification = classificationResult[dir.Path]
		}
	}

	return c.Status(fiber.StatusOK).JSON(&fileSystemResponse{
		Count:       len(drives),
		Directories: directories,
		Files:       []*fileInfoResponse{},
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// path queries the path and builds a slice of files and directories
func (api fsAPI) path(c *fiber.Ctx) error {
	// Verify authentication first
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	encodedPath := c.Params("path")

	path, err := utils.DecodeString(encodedPath)
	if err != nil {
		return errorResponse(c, fiber.StatusBadRequest, "Invalid path encoding", nil)
	}

	directories := make([]*fileInfoResponse, 0)
	files := make([]*fileInfoResponse, 0)

	// Get a string slice of items in a directory
	items, err := api.r.app.FS.ReadDir(path, true)
	if err != nil {
		return errorResponse(c, fiber.StatusNotFound, "Error reading directory", nil)
	}

	paths := make([]string, 0)
	for _, directory := range items.Directories {
		path := utils.NormalizeWindowsDrive(filepath.Join(path, directory.Name()))
		paths = append(paths, path)

		directories = append(directories, &fileInfoResponse{Title: directory.Name(), Path: path})
	}

	// Include path classification; ancestor, course, descendant, none
	if classificationResult, err := api.r.appDao.ClassifyCoursePaths(ctx, paths); err != nil {
		return errorResponse(c, fiber.StatusInternalServerError, "Error classifying paths", nil)
	} else {
		for _, dir := range directories {
			dir.Classification = classificationResult[dir.Path]
		}
	}

	for _, file := range items.Files {
		files = append(files, &fileInfoResponse{Title: file.Name(), Path: filepath.Join(path, file.Name())})
	}

	return c.Status(fiber.StatusOK).JSON(&fileSystemResponse{
		Count:       len(directories) + len(files),
		Directories: directories,
		Files:       files,
	})
}
