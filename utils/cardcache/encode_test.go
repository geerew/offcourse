package cardcache

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/image/tiff"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestEncodeImage(t *testing.T) {
	t.Run("JPEG", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 1200, 600))
		for y := range 600 {
			for x := range 1200 {
				img.Set(x, y, color.RGBA{
					R: uint8(x % 256),
					G: uint8(y % 256),
					B: 128,
					A: 255,
				})
			}
		}

		var src bytes.Buffer
		require.NoError(t, jpeg.Encode(&src, img, &jpeg.Options{Quality: 90}))

		out, err := encodeImageToWebP(src.Bytes())
		require.NoError(t, err)
		require.NotEmpty(t, out)
		require.Less(t, len(out), src.Len())
	})

	t.Run("TIFF", func(t *testing.T) {
		img := image.NewRGBA(image.Rect(0, 0, 200, 200))
		for y := range 200 {
			for x := range 200 {
				img.Set(x, y, color.RGBA{R: 10, G: 20, B: 30, A: 255})
			}
		}

		var src bytes.Buffer
		require.NoError(t, tiff.Encode(&src, img, nil))

		out, err := encodeImageToWebP(src.Bytes())
		require.NoError(t, err)
		require.NotEmpty(t, out)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully scaling an image to a maximum width
func TestScaleToMaxWidth(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1600, 400))
	scaled := scaleToMaxWidth(img, maxCardWidth)
	require.Equal(t, maxCardWidth, scaled.Bounds().Dx())
	require.Equal(t, 200, scaled.Bounds().Dy())
}
