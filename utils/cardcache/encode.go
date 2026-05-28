package cardcache

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/chai2010/webp"
	"golang.org/x/image/draw"
	_ "golang.org/x/image/tiff"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	maxCardWidth = 800
	webpQuality  = 85
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// encodeImageToWebP decodes image bytes, scales to maxCardWidth, and lossy-encodes as WebP.
func encodeImageToWebP(data []byte) ([]byte, error) {
	img, err := decodeImage(data)
	if err != nil {
		return nil, err
	}

	scaled := scaleToMaxWidth(img, maxCardWidth)

	var buf bytes.Buffer
	if err := webp.Encode(&buf, scaled, &webp.Options{Quality: webpQuality}); err != nil {
		return nil, fmt.Errorf("failed to encode webp: %w", err)
	}

	return buf.Bytes(), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// decodeImage decodes image bytes into a image.Image
func decodeImage(data []byte) (image.Image, error) {
	if img, err := webp.Decode(bytes.NewReader(data)); err == nil {
		return img, nil
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	return img, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// scaleToMaxWidth scales an image to a maximum width
func scaleToMaxWidth(src image.Image, maxWidth int) image.Image {
	bounds := src.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= maxWidth {
		return src
	}

	newWidth := maxWidth
	newHeight := height * maxWidth / width
	if newHeight < 1 {
		newHeight = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	return dst
}
