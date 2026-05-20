package cardcache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/concurrency"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/media"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ErrInvalidCourseID is returned when a course ID is unsafe for cache file paths
var ErrInvalidCourseID = errors.New("invalid course id")

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CardCache manages optimized card image generation and caching
type CardCache struct {
	config     *CardCacheConfig
	cachePath  string
	serveIndex concurrency.Map[string, CardServe]
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CardCacheConfig defines the configuration for a CardCache
type CardCacheConfig struct {
	CachePath string
	AppFs     *appfs.AppFs
	Logger    *logger.Logger
	FFmpeg    *media.FFmpeg
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// New creates a new CardCache and prepares the cache directory
func New(config *CardCacheConfig) (*CardCache, error) {
	if config == nil {
		return nil, fmt.Errorf("card cache config is nil")
	}

	if config.AppFs == nil || config.AppFs.Fs == nil {
		return nil, fmt.Errorf("card cache app filesystem is nil")
	}

	if config.Logger == nil {
		return nil, fmt.Errorf("card cache logger is nil")
	}

	if config.CachePath == "" {
		return nil, fmt.Errorf("card cache path is empty")
	}

	var cachePath string
	if _, ok := config.AppFs.Fs.(*afero.MemMapFs); ok {
		cachePath = filepath.Join(config.CachePath, "cards")
	} else {
		absDataDir, err := filepath.Abs(config.CachePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for cache path: %w", err)
		}

		cachePath = filepath.Join(absDataDir, "cards")
	}

	if err := config.AppFs.Fs.MkdirAll(cachePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create card cache directory: %w", err)
	}

	cardCache := &CardCache{
		config:     config,
		cachePath:  cachePath,
		serveIndex: concurrency.NewMap[string, CardServe](),
	}

	outputPath := cardCache.fallbackPath()

	if err := afero.WriteFile(cardCache.config.AppFs.Fs, outputPath, fallbackCardBytes, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write fallback card: %w", err)
	}

	return cardCache, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// OptimizeCard generates an optimized WebP when smaller than the source and updates the in-memory
// serve index to point at the optimized file, the original, or the fallback image
func (c *CardCache) OptimizeCard(ctx context.Context, courseID, originalPath string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if c.config.FFmpeg == nil {
		return fmt.Errorf("ffmpeg is not configured")
	}

	optimizedCardPath, err := c.optimizedCardPath(courseID)
	if err != nil {
		return err
	}

	originalInfo, err := c.config.AppFs.Fs.Stat(originalPath)
	if err != nil {
		c.setServeOriginalOrFallback(courseID, originalPath)
		return fmt.Errorf("failed to stat original card: %w", err)
	}

	outputDir := filepath.Dir(optimizedCardPath)
	err = c.config.AppFs.Fs.MkdirAll(outputDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	args := []string{
		"-nostats",
		"-hide_banner",
		"-loglevel", "warning",
		"-i", originalPath,
		"-vf", "scale=800:-1",
		"-c:v", "libwebp",
		"-quality", "85",
		"-y",
		optimizedCardPath,
	}

	cmd := exec.CommandContext(ctx, c.config.FFmpeg.GetFFmpegPath(), args...)

	c.config.Logger.Debug().
		Str("original_path", originalPath).
		Str("output_path", optimizedCardPath).
		Str("command", strings.Join(cmd.Args, " ")).
		Msg("Running FFmpeg for card optimization")

	var stderr strings.Builder
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		_ = c.deleteCard(optimizedCardPath)

		if ctx.Err() != nil {
			return ctx.Err()
		}

		c.config.Logger.Error().
			Err(err).
			Str("course_id", courseID).
			Str("original_path", originalPath).
			Str("output_path", optimizedCardPath).
			Str("stderr", stderr.String()).
			Msg("Failed to generate optimized card")

		c.setServeOriginalOrFallback(courseID, originalPath)
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	optimizedInfo, err := c.config.AppFs.Fs.Stat(optimizedCardPath)
	if err != nil {
		_ = c.deleteCard(optimizedCardPath)
		c.setServeOriginalOrFallback(courseID, originalPath)
		return fmt.Errorf("failed to stat optimized card: %w", err)
	}

	if optimizedInfo.Size() >= originalInfo.Size() {
		_ = c.deleteCard(optimizedCardPath)

		c.config.Logger.Info().
			Str("course_id", courseID).
			Str("original_path", originalPath).
			Str("output_path", optimizedCardPath).
			Int64("original_bytes", originalInfo.Size()).
			Int64("optimized_bytes", optimizedInfo.Size()).
			Msg("Skipped card cache; optimized file is not smaller than original")

		c.setServeOriginal(courseID, originalPath)
		return nil
	}

	c.config.Logger.Info().
		Str("course_id", courseID).
		Str("original_path", originalPath).
		Str("output_path", optimizedCardPath).
		Int64("original_bytes", originalInfo.Size()).
		Int64("optimized_bytes", optimizedInfo.Size()).
		Msg("Generated optimized card")

	if err := c.setServeOptimized(courseID); err != nil {
		return err
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// deleteCard deletes a card from the cache directory
func (c *CardCache) deleteCard(cardPath string) error {
	err := c.config.AppFs.Fs.Remove(cardPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to delete card: %w", err)
	}

	c.config.Logger.Debug().
		Str("card_path", cardPath).
		Msg("Deleted optimized card")

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// cardExists checks if a card exists at the given path
func (c *CardCache) cardExists(cardPath string) (bool, error) {
	exists, err := afero.Exists(c.config.AppFs.Fs, cardPath)
	if err != nil {
		return false, fmt.Errorf("failed to check if card exists: %w", err)
	}

	return exists, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// optimizedCardPath returns a joined path to the optimized card. It will error
// if the course ID is empty, contains path traversal, or contains path separators
func (c *CardCache) optimizedCardPath(courseID string) (string, error) {
	if courseID == "" || strings.Contains(courseID, "..") || strings.ContainsAny(courseID, `/\`) {
		return "", ErrInvalidCourseID
	}

	return filepath.Join(c.cachePath, courseID+".webp"), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// fallbackPath returns a joined path to the fallback card
func (c *CardCache) fallbackPath() string {
	return filepath.Join(c.cachePath, "fallback.webp")
}
