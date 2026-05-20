package cardcache

import (
	"errors"
	"mime"
	"path/filepath"

	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ErrFallbackNotFound is returned when the fallback card file is missing
var ErrFallbackNotFound = errors.New("fallback card not found")

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CardServe is a resolved card path and metadata for HTTP serving
type CardServe struct {
	Path        string
	ContentType string
	Fallback    bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseCardRef identifies a course card for warming the serve index
type CourseCardRef struct {
	ID       string
	CardPath string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Get returns the in-memory serve entry for a course, or the fallback image when
// none is set
func (c *CardCache) Get(courseID string) (CardServe, error) {
	if _, err := c.optimizedCardPath(courseID); err != nil {
		return c.fallbackServe()
	}

	if serve, ok := c.serveIndex.Get(courseID); ok {
		exists, err := afero.Exists(c.config.AppFs.Fs, serve.Path)
		if err != nil {
			return CardServe{}, err
		}

		if exists {
			return serve, nil
		}

		c.serveIndex.Remove(courseID)
	}

	return c.fallbackServe()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Delete removes the optimized cache file and sets the serve entry to the fallback
// image
func (c *CardCache) Delete(courseID string) error {
	if optimizedPath, err := c.optimizedCardPath(courseID); err == nil {
		if err := c.deleteCard(optimizedPath); err != nil {
			return err
		}
	}

	c.setServeFallback(courseID)
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Warm rebuilds the in-memory serve index from disk for each course (optimized WebP,
// original source, or fallback). Call on startup so Get works before the next scan.
func (c *CardCache) Warm(refs []CourseCardRef) int {
	warmed := 0

	for _, ref := range refs {
		if ref.ID == "" {
			continue
		}

		serve, err := c.resolveServeFromDisk(ref.ID, ref.CardPath)
		if err != nil {
			continue
		}

		c.serveIndex.Set(ref.ID, serve)
		warmed++
	}

	return warmed
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setServeOptimized sets the serve entry to the optimized card
func (c *CardCache) setServeOptimized(courseID string) error {
	optimizedPath, err := c.optimizedCardPath(courseID)
	if err != nil {
		return err
	}

	c.serveIndex.Set(courseID, CardServe{
		Path:        optimizedPath,
		ContentType: "image/webp",
	})

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setServeOriginal sets the serve entry to the original card
func (c *CardCache) setServeOriginal(courseID, originalPath string) {
	c.serveIndex.Set(courseID, c.cardServeOriginal(originalPath))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// cardServeOriginal returns a CardServe entry for the original card
func (c *CardCache) cardServeOriginal(originalPath string) CardServe {
	contentType := mime.TypeByExtension(filepath.Ext(originalPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return CardServe{
		Path:        originalPath,
		ContentType: contentType,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setServeFallback sets the serve entry to the fallback card
func (c *CardCache) setServeFallback(courseID string) {
	serve, err := c.fallbackServe()
	if err != nil {
		return
	}

	c.serveIndex.Set(courseID, serve)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (c *CardCache) resolveServeFromDisk(courseID, originalPath string) (CardServe, error) {
	if _, err := c.optimizedCardPath(courseID); err != nil {
		return c.fallbackServe()
	}

	if optimizedPath, err := c.optimizedCardPath(courseID); err == nil {
		exists, err := c.cardExists(optimizedPath)
		if err != nil {
			return CardServe{}, err
		}

		if exists {
			return CardServe{
				Path:        optimizedPath,
				ContentType: "image/webp",
			}, nil
		}
	}

	if originalPath != "" {
		exists, err := afero.Exists(c.config.AppFs.Fs, originalPath)
		if err != nil {
			return CardServe{}, err
		}

		if exists {
			return c.cardServeOriginal(originalPath), nil
		}
	}

	return c.fallbackServe()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setServeOriginalOrFallback sets the serve entry to the original card or the
// fallback card
func (c *CardCache) setServeOriginalOrFallback(courseID, originalPath string) {
	if originalPath != "" {
		exists, err := afero.Exists(c.config.AppFs.Fs, originalPath)
		if err == nil && exists {
			c.setServeOriginal(courseID, originalPath)
			return
		}
	}

	c.setServeFallback(courseID)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// fallbackServe returns the fallback card serve entry
func (c *CardCache) fallbackServe() (CardServe, error) {
	fallbackPath := c.fallbackPath()

	exists, err := c.cardExists(fallbackPath)
	if err != nil {
		return CardServe{}, err
	}

	if !exists {
		return CardServe{}, ErrFallbackNotFound
	}

	return CardServe{
		Path:        fallbackPath,
		ContentType: "image/webp",
		Fallback:    true,
	}, nil
}
