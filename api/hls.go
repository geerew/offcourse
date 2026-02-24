package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/media/hls"
	"github.com/gofiber/fiber/v2"
	"github.com/houseme/mobiledetect/ua"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// hlsAPI handles HLS transcoding requests
type hlsAPI struct {
	r *Router
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// RegisterHLSRoutes registers HLS routes
func (r *Router) initHlsRoutes() {

	hlsApi := hlsAPI{r: r}

	g := r.apiGroup("hls")

	// Master playlist
	g.Get("/:asset_id/master.m3u8", hlsApi.GetMaster)

	// Video streams
	g.Get("/:asset_id/video/:index/:quality/index.m3u8", hlsApi.GetVideoIndex)
	g.Get("/:asset_id/video/:index/:quality/segment-:num.ts", hlsApi.GetVideoSegment)

	// Audio streams
	g.Get("/:asset_id/audio/:index/index.m3u8", hlsApi.GetAudioIndex)
	g.Get("/:asset_id/audio/:index/segment-:num.ts", hlsApi.GetAudioSegment)

	// Qualities endpoint
	g.Get("/:asset_id/qualities", hlsApi.GetQualities)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetMaster returns the master playlist (single stream based on device type)
func (api *hlsAPI) GetMaster(c *fiber.Ctx) error {
	assetID := c.Params("asset_id")
	if assetID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "asset_id is required",
		})
	}

	// Verify authentication
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Get asset with metadata and verify it belongs to a course
	asset, err := api.getAssetWithMetadataAndCourse(ctx, assetID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}
	if asset == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}

	// Check if transcoder is available
	if api.r.app.Transcoder == nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Transcoder not available",
		})
	}

	ua := ua.New(c.Get("User-Agent"))

	// Get simple master playlist (single stream based on device type)
	master, err := api.r.app.Transcoder.GetMasterPlaylistSingle(ctx, asset.Path, assetID, ua.Mobile())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate master playlist",
		})
	}

	c.Set("Content-Type", "application/vnd.apple.mpegurl")
	return c.Status(http.StatusOK).SendString(master)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetVideoIndex returns the video index playlist
func (api *hlsAPI) GetVideoIndex(c *fiber.Ctx) error {
	assetID := c.Params("asset_id")
	indexStr := c.Params("index")
	qualityStr := c.Params("quality")

	index, err := strconv.ParseUint(indexStr, 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid video index",
		})
	}

	quality, err := hls.QualityFromString(qualityStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid quality",
		})
	}

	// Verify authentication
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Get asset and verify it belongs to a course
	asset, err := api.getAssetWithMetadataAndCourse(ctx, assetID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}
	if asset == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}

	// Get video index
	indexPlaylist, err := api.r.app.Transcoder.GetVideoIndex(ctx, asset.Path, uint32(index), quality, assetID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate video index",
		})
	}

	c.Set("Content-Type", "application/vnd.apple.mpegurl")
	return c.Status(http.StatusOK).SendString(indexPlaylist)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetVideoSegment returns a video segment
func (api *hlsAPI) GetVideoSegment(c *fiber.Ctx) error {
	assetID := c.Params("asset_id")
	indexStr := c.Params("index")
	qualityStr := c.Params("quality")
	segmentStr := c.Params("num")

	index, err := strconv.ParseUint(indexStr, 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid video index",
		})
	}

	quality, err := hls.QualityFromString(qualityStr)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid quality",
		})
	}

	segment, err := strconv.ParseInt(segmentStr, 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid segment number",
		})
	}

	// Verify authentication
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Get asset and verify it belongs to a course
	asset, err := api.getAssetWithMetadataAndCourse(ctx, assetID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}
	if asset == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}

	// Get video segment
	segmentPath, err := api.r.app.Transcoder.GetVideoSegment(ctx, asset.Path, uint32(index), quality, int32(segment), assetID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate video segment",
		})
	}

	// Serve the segment file
	return c.Status(http.StatusOK).SendFile(segmentPath)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAudioIndex returns the audio index playlist
func (api *hlsAPI) GetAudioIndex(c *fiber.Ctx) error {
	assetID := c.Params("asset_id")
	indexStr := c.Params("index")

	index, err := strconv.ParseUint(indexStr, 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid audio index",
		})
	}

	// Verify authentication
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Get asset and verify it belongs to a course
	asset, err := api.getAssetWithMetadataAndCourse(ctx, assetID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}
	if asset == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}

	// Get audio index
	indexPlaylist, err := api.r.app.Transcoder.GetAudioIndex(ctx, asset.Path, uint32(index), assetID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate audio index",
		})
	}

	c.Set("Content-Type", "application/vnd.apple.mpegurl")
	return c.Status(http.StatusOK).SendString(indexPlaylist)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAudioSegment returns an audio segment
func (api *hlsAPI) GetAudioSegment(c *fiber.Ctx) error {
	assetID := c.Params("asset_id")
	indexStr := c.Params("index")
	segmentStr := c.Params("num")

	index, err := strconv.ParseUint(indexStr, 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid audio index",
		})
	}

	segment, err := strconv.ParseInt(segmentStr, 10, 32)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid segment number",
		})
	}

	// Verify authentication
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Get asset and verify it belongs to a course
	asset, err := api.getAssetWithMetadataAndCourse(ctx, assetID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}
	if asset == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}

	// Get audio segment
	segmentPath, err := api.r.app.Transcoder.GetAudioSegment(ctx, asset.Path, uint32(index), int32(segment), assetID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate audio segment",
		})
	}

	// Serve the segment file
	return c.Status(http.StatusOK).SendFile(segmentPath)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetQualities returns the available qualities for a video
func (api *hlsAPI) GetQualities(c *fiber.Ctx) error {
	assetID := c.Params("asset_id")
	if assetID == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error": "asset_id is required",
		})
	}

	// Verify authentication
	_, ctx, err := principalCtx(c)
	if err != nil {
		return errorResponse(c, fiber.StatusUnauthorized, "Missing principal", nil)
	}

	// Get asset with metadata and verify it belongs to a course
	asset, err := api.getAssetWithMetadataAndCourse(ctx, assetID)
	if err != nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}
	if asset == nil {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error": "Asset not found",
		})
	}

	// Check if transcoder is available
	if api.r.app.Transcoder == nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Transcoder not available",
		})
	}

	// Get available qualities
	qualities, err := api.r.app.Transcoder.GetQualities(ctx, asset.Path, assetID)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get available qualities",
		})
	}

	// Convert qualities to strings for JSON response
	qualityStrings := make([]string, len(qualities))
	for i, quality := range qualities {
		qualityStrings[i] = string(quality)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"qualities": qualityStrings,
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getAssetWithMetadataAndCourse retrieves an asset with its metadata by asset ID
// and verifies it belongs to a course
func (api *hlsAPI) getAssetWithMetadataAndCourse(ctx context.Context, assetID string) (*models.Asset, error) {
	dbOpts := dao.NewOptions().
		WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: assetID}).
		WithAssetMetadata()

	asset, err := api.r.appDao.GetAsset(ctx, dbOpts)
	if err != nil {
		return nil, err
	}

	// Verify asset exists and belongs to a course
	if asset == nil || asset.CourseID == "" {
		return nil, nil
	}

	return asset, nil
}
