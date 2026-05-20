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
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/media"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ErrInvalidCourseID is returned when a course ID is unsafe for cache file paths.
var ErrInvalidCourseID = errors.New("invalid course id")

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CardCache manages optimized card image generation and caching
type CardCache struct {
	config    *CardCacheConfig
	cachePath string
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

	absDataDir, err := filepath.Abs(config.CachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for cache path: %w", err)
	}

	cachePath := filepath.Join(absDataDir, "cards")

	err = config.AppFs.Fs.MkdirAll(cachePath, 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to create card cache directory: %w", err)
	}

	cardCache := &CardCache{
		config:    config,
		cachePath: cachePath,
	}

	return cardCache, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GenerateOptimizedCard generates an optimized WebP version of a card image
func (c *CardCache) GenerateOptimizedCard(ctx context.Context, originalPath, outputPath string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if c.config.FFmpeg == nil {
		return fmt.Errorf("ffmpeg is not configured")
	}

	// Ensure output directory exists
	outputDir := filepath.Dir(outputPath)
	err := c.config.AppFs.Fs.MkdirAll(outputDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build FFmpeg command for image conversion
	args := []string{
		"-nostats",
		"-hide_banner",
		"-loglevel", "warning",
		"-i", originalPath,
		"-vf", "scale=800:-1",
		"-c:v", "libwebp",
		"-quality", "85",
		"-y",
		outputPath,
	}

	// Create command with context for cancellation support
	cmd := exec.CommandContext(ctx, c.config.FFmpeg.GetFFmpegPath(), args...)

	c.config.Logger.Debug().
		Str("original_path", originalPath).
		Str("output_path", outputPath).
		Str("command", strings.Join(cmd.Args, " ")).
		Msg("Running FFmpeg for card optimization")

	// Capture stderr for error reporting
	var stderr strings.Builder
	cmd.Stderr = &stderr

	// Run the command
	err = cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		c.config.Logger.Error().
			Err(err).
			Str("original_path", originalPath).
			Str("output_path", outputPath).
			Str("stderr", stderr.String()).
			Msg("Failed to generate optimized card")

		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	c.config.Logger.Debug().
		Str("original_path", originalPath).
		Str("output_path", outputPath).
		Msg("Successfully generated optimized card")

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// EnsureFallbackCard writes the embedded fallback WebP to the cache directory
func (c *CardCache) EnsureFallbackCard(outputPath string) error {
	outputDir := filepath.Dir(outputPath)
	err := c.config.AppFs.Fs.MkdirAll(outputDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	err = afero.WriteFile(c.config.AppFs.Fs, outputPath, fallbackCardBytes, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write fallback card: %w", err)
	}

	c.config.Logger.Debug().
		Str("output_path", outputPath).
		Msg("Ensured fallback card exists")

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteCard deletes an optimized card file
func (c *CardCache) DeleteCard(cardPath string) error {
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

// CardExists checks if an optimized card exists
func (c *CardCache) CardExists(cardPath string) (bool, error) {
	exists, err := afero.Exists(c.config.AppFs.Fs, cardPath)
	if err != nil {
		return false, fmt.Errorf("failed to check if card exists: %w", err)
	}

	return exists, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetCardPath returns the full path to an optimized card for a course
func (c *CardCache) GetCardPath(courseID string) (string, error) {
	if courseID == "" || strings.Contains(courseID, "..") || strings.ContainsAny(courseID, `/\`) {
		return "", ErrInvalidCourseID
	}

	return filepath.Join(c.cachePath, courseID+".webp"), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetFallbackPath returns the full path to the fallback card
func (c *CardCache) GetFallbackPath() string {
	return filepath.Join(c.cachePath, "fallback.webp")
}
