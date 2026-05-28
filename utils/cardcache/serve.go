package cardcache

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"path/filepath"

	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ErrFallbackNotFound is returned when the fallback card file is missing
var ErrFallbackNotFound = errors.New("fallback card not found")

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CardServe is resolved card metadata for HTTP serving
type CardServe struct {
	Path        string
	ContentType string
	CardHash    string
	Fallback    bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CourseCardRef identifies a course card for warming the serve index
type CourseCardRef struct {
	ID       string
	CardPath string
	CardHash string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Get returns the in-memory serve entry for a course, or the fallback image when
// none is set
func (c *CardCache) Get(courseID string) (CardServe, error) {
	if _, err := c.optimizedCardPath(courseID); err != nil {
		return c.fallbackServe()
	}

	if serve, ok := c.serveIndex.Get(courseID); ok {
		exists, err := afero.Exists(c.config.FS, serve.Path)
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

// OpenCard resolves the card for a course and returns a reader for its bytes.
func (c *CardCache) OpenCard(courseID string) (io.ReadCloser, CardServe, error) {
	serve, err := c.Get(courseID)
	if err != nil {
		return nil, CardServe{}, err
	}

	f, err := c.config.FS.Open(serve.Path)
	if err != nil {
		return nil, CardServe{}, fmt.Errorf("failed to open card: %w", err)
	}

	return f, serve, nil
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

		serve, err := c.resolveServeFromDisk(ref.ID, ref.CardPath, ref.CardHash)
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
func (c *CardCache) setServeOptimized(courseID, cardHash string) error {
	optimizedPath, err := c.optimizedCardPath(courseID)
	if err != nil {
		return err
	}

	c.serveIndex.Set(courseID, CardServe{
		Path:        optimizedPath,
		ContentType: "image/webp",
		CardHash:    cardHash,
	})

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setServeOriginal sets the serve entry to the original card
func (c *CardCache) setServeOriginal(courseID, originalPath, cardHash string) {
	c.serveIndex.Set(courseID, c.cardServeOriginal(originalPath, cardHash))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// cardServeOriginal returns a CardServe entry for the original card
func (c *CardCache) cardServeOriginal(originalPath, cardHash string) CardServe {
	contentType := mime.TypeByExtension(filepath.Ext(originalPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return CardServe{
		Path:        originalPath,
		ContentType: contentType,
		CardHash:    cardHash,
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

func (c *CardCache) resolveServeFromDisk(courseID, originalPath, cardHash string) (CardServe, error) {
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
				CardHash:    cardHash,
			}, nil
		}
	}

	if originalPath != "" {
		exists, err := afero.Exists(c.config.FS, originalPath)
		if err != nil {
			return CardServe{}, err
		}

		if exists {
			return c.cardServeOriginal(originalPath, cardHash), nil
		}
	}

	return c.fallbackServe()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setServeOriginalOrFallback sets the serve entry to the original card or the
// fallback card
func (c *CardCache) setServeOriginalOrFallback(courseID, originalPath, cardHash string) {
	if originalPath != "" {
		exists, err := afero.Exists(c.config.FS, originalPath)
		if err == nil && exists {
			c.setServeOriginal(courseID, originalPath, cardHash)
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
