package dao

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func helper_createAssetMetadata(t *testing.T, ctx context.Context, dao *DAO, count int) ([]*models.Asset, []*models.AssetMetadata) {
	t.Helper()

	// Course
	course := &models.Course{Title: "Course 1", Path: "/course-1"}
	require.NoError(t, dao.CreateCourse(ctx, course))

	// Lesson
	lesson := &models.Lesson{
		CourseID: course.ID,
		Title:    "Asset Group 1",
		Prefix:   sql.NullInt16{Int16: 1, Valid: true},
		Module:   "Module 1",
	}
	require.NoError(t, dao.CreateLesson(ctx, lesson))

	assets := make([]*models.Asset, 0, count)
	assetsMetadata := make([]*models.AssetMetadata, 0, count)

	for i := 0; i < count; i++ {
		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    fmt.Sprintf("Asset %d", i+1),
			Prefix:   sql.NullInt16{Int16: int16(i + 1), Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     fmt.Sprintf("/course-1/0%d asset.mp4", i+1),
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     fmt.Sprintf("hash-%d", i+1),
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))
		assets = append(assets, asset)

		video := &models.VideoMetadata{
			DurationSec: i + 1,
			Container:   "mov,mp4,m4a,3gp,3g2,mj2",
			MIMEType:    "video/mp4",
			SizeBytes:   1024,
			OverallBPS:  200_000,
			VideoCodec:  "h264",
			Width:       1280,
			Height:      720,
			FPSNum:      30,
			FPSDen:      1,
		}

		audio := &models.AudioMetadata{
			Language:      "und",
			Codec:         "aac",
			Profile:       "LC",
			Channels:      1,
			ChannelLayout: "mono",
			SampleRate:    48000,
			BitRate:       128_000,
		}

		assetMetadata := &models.AssetMetadata{
			AssetID:       asset.ID,
			VideoMetadata: video,
			AudioMetadata: audio,
		}

		require.NoError(t, dao.CreateAssetMetadata(ctx, assetMetadata))
		assetsMetadata = append(assetsMetadata, assetMetadata)

		time.Sleep(1 * time.Millisecond)
	}

	return assets, assetsMetadata
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_CreateVideoMetadata(t *testing.T) {
	t.Run("success (video only)", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     filepath.ToSlash("/course-1/01 asset.mp4"),
			FileSize: 1024,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "1234",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		meta := &models.AssetMetadata{
			AssetID: asset.ID,
			VideoMetadata: &models.VideoMetadata{
				DurationSec: 120,
				Container:   "mov,mp4,m4a,3gp,3g2,mj2",
				MIMEType:    "video/mp4",
				SizeBytes:   1024,
				OverallBPS:  200000,
				VideoCodec:  "h264",
				Width:       1280,
				Height:      720,
				FPSNum:      30,
				FPSDen:      1,
			},
			AudioMetadata: nil,
		}
		require.NoError(t, dao.CreateAssetMetadata(ctx, meta))

		record, err := dao.GetAssetMetadata(ctx, asset.ID)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.NotNil(t, record.VideoMetadata)
		require.Nil(t, record.AudioMetadata)

		require.Equal(t, 120, record.VideoMetadata.DurationSec)
		require.Equal(t, "video/mp4", record.VideoMetadata.MIMEType)
		require.Equal(t, "h264", record.VideoMetadata.VideoCodec)
		require.Equal(t, 1280, record.VideoMetadata.Width)
		require.Equal(t, 720, record.VideoMetadata.Height)
		require.Equal(t, 30, record.VideoMetadata.FPSNum)
		require.Equal(t, 1, record.VideoMetadata.FPSDen)
	})

	t.Run("success (video + audio)", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 2", Path: "/course-2"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 2",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 2",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset 2",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 2",
			Type:     types.MustAsset("mp4"),
			Path:     filepath.ToSlash("/course-2/02 asset.mp4"),
			FileSize: 2048,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "5678",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		meta := &models.AssetMetadata{
			AssetID: asset.ID,
			VideoMetadata: &models.VideoMetadata{
				DurationSec: 5,
				Container:   "mov,mp4,m4a,3gp,3g2,mj2",
				MIMEType:    "video/mp4",
				SizeBytes:   2048,
				OverallBPS:  250000,
				VideoCodec:  "h264",
				Width:       1280,
				Height:      720,
				FPSNum:      30,
				FPSDen:      1,
			},
			AudioMetadata: &models.AudioMetadata{
				Language:      "und",
				Codec:         "aac",
				Profile:       "LC",
				Channels:      1,
				ChannelLayout: "mono",
				SampleRate:    48000,
				BitRate:       128000,
			},
		}
		require.NoError(t, dao.CreateAssetMetadata(ctx, meta))

		record, err := dao.GetAssetMetadata(ctx, asset.ID)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.NotNil(t, record.VideoMetadata)
		require.NotNil(t, record.AudioMetadata)

		require.Equal(t, "video/mp4", record.VideoMetadata.MIMEType)
		require.Equal(t, "aac", record.AudioMetadata.Codec)
		require.Equal(t, 48000, record.AudioMetadata.SampleRate)
		require.GreaterOrEqual(t, record.AudioMetadata.Channels, 1)
	})

	t.Run("nil input", func(t *testing.T) {
		dao, ctx := setup(t)
		require.ErrorIs(t, dao.CreateAssetMetadata(ctx, nil), utils.ErrNilPtr)
	})

	t.Run("nil metadata", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course 3", Path: "/course-3"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 3",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 3",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset 3",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 3",
			Type:     types.MustAsset("md"),
			Path:     filepath.ToSlash("/course-3/03 asset.md"),
			FileSize: 100,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "9999",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		meta := &models.AssetMetadata{
			AssetID:       asset.ID,
			VideoMetadata: nil,
			AudioMetadata: nil,
		}
		require.NoError(t, dao.CreateAssetMetadata(ctx, meta))

		record, err := dao.GetAssetMetadata(ctx, asset.ID)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetAssetMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 1)

		record, err := dao.GetAssetMetadata(ctx, assets[0].ID)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.NotNil(t, record.VideoMetadata)
		require.NotNil(t, record.AudioMetadata)

		require.Equal(t, 1280, record.VideoMetadata.Width)
		require.Equal(t, 720, record.VideoMetadata.Height)
		require.Equal(t, "h264", record.VideoMetadata.VideoCodec)
		require.Equal(t, "video/mp4", record.VideoMetadata.MIMEType)
		require.Equal(t, 30, record.VideoMetadata.FPSNum)
		require.Equal(t, 1, record.VideoMetadata.FPSDen)

		require.Equal(t, "aac", record.AudioMetadata.Codec)
		require.Equal(t, 48000, record.AudioMetadata.SampleRate)
		require.Equal(t, 1, record.AudioMetadata.Channels)
	})

	t.Run("no metadata", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course NM", Path: "/course-nm"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Group NM",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module NM",
		}
		require.NoError(t, dao.CreateLesson(ctx, lesson))

		asset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset NM",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module NM",
			Type:     types.MustAsset("md"),
			Path:     filepath.ToSlash("/course-nm/01 asset.md"),
			FileSize: 100,
			ModTime:  time.Now().Format(time.RFC3339Nano),
			Hash:     "no-meta",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		record, err := dao.GetAssetMetadata(ctx, asset.ID)
		require.NoError(t, err)
		require.Nil(t, record)
	})

	t.Run("asset id not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetAssetMetadata(ctx, "does-not-exist")
		require.NoError(t, err)
		require.Nil(t, record)

	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListAssetMetadata(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 3)

		records, err := dao.ListAssetMetadata(ctx, nil)
		require.NoError(t, err)
		require.NotNil(t, records)
		require.Len(t, records, 3)

		// Ensure each result has some metadata and matches one of the created assets
		recordsMap := make(map[string]bool, len(records))
		for _, r := range records {
			require.NotNil(t, r)
			require.NotEmpty(t, r.AssetID)
			require.True(t, r.VideoMetadata != nil || r.AudioMetadata != nil)
			recordsMap[r.AssetID] = true
		}

		for _, a := range assets {
			require.True(t, recordsMap[a.ID], "missing asset metadata for asset id %s", a.ID)
		}
	})

	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListAssetMetadata(ctx, nil)
		require.NoError(t, err)
		require.Nil(t, records)
	})

	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 3)

		// Video created_at ascending
		opts := NewOptions().
			WithOrderBy(models.ASSET_METADATA_VIDEO_TABLE_CREATED_AT + " ASC")

		records, err := dao.ListAssetMetadata(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, records)
		require.Len(t, records, 3)

		require.Equal(t, assets[0].ID, records[0].AssetID)
		require.Equal(t, assets[1].ID, records[1].AssetID)
		require.Equal(t, assets[2].ID, records[2].AssetID)

		// Video created_at DESC
		opts = NewOptions().
			WithOrderBy(models.ASSET_METADATA_VIDEO_TABLE_CREATED_AT + " DESC")

		records, err = dao.ListAssetMetadata(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, records)
		require.Len(t, records, 3)

		require.Equal(t, assets[2].ID, records[0].AssetID)
		require.Equal(t, assets[1].ID, records[1].AssetID)
		require.Equal(t, assets[0].ID, records[2].AssetID)
	})

	t.Run("where (single asset)", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 3)

		opts := NewOptions().
			WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: assets[1].ID})

		records, err := dao.ListAssetMetadata(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, records)
		require.Len(t, records, 1)
		require.Equal(t, assets[1].ID, records[0].AssetID)
	})

	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 17)

		// Make ordering deterministic for test
		opts := NewOptions().
			WithOrderBy(models.ASSET_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(1, 10))

		// Page 1
		page1, err := dao.ListAssetMetadata(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, page1)
		require.Len(t, page1, 10)
		require.Equal(t, assets[0].ID, page1[0].AssetID)
		require.Equal(t, assets[9].ID, page1[9].AssetID)

		// Page 2
		opts = NewOptions().
			WithOrderBy(models.ASSET_TABLE_CREATED_AT + " ASC").
			WithPagination(pagination.New(2, 10))

		page2, err := dao.ListAssetMetadata(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, page2)
		require.Len(t, page2, 7)
		require.Equal(t, assets[10].ID, page2[0].AssetID)
		require.Equal(t, assets[16].ID, page2[6].AssetID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpdateAssetMetadata(t *testing.T) {
	t.Run("no-op", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 1)
		assetID := assets[0].ID

		// Snapshot existing
		before, err := dao.GetAssetMetadata(ctx, assetID)
		require.NoError(t, err)
		require.NotNil(t, before)
		require.NotNil(t, before.VideoMetadata)
		require.NotNil(t, before.AudioMetadata)

		time.Sleep(2 * time.Millisecond)

		// No-op update
		err = dao.UpdateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID:       assetID,
			VideoMetadata: nil,
			AudioMetadata: nil,
		})
		require.NoError(t, err)

		after, err := dao.GetAssetMetadata(ctx, assetID)
		require.NoError(t, err)
		require.NotNil(t, after)
		require.NotNil(t, after.VideoMetadata)
		require.NotNil(t, after.AudioMetadata)

		require.True(t, after.VideoMetadata.UpdatedAt.Equal(before.VideoMetadata.UpdatedAt))
		require.True(t, after.AudioMetadata.UpdatedAt.Equal(before.AudioMetadata.UpdatedAt))
	})

	t.Run("video update", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, beforeAll := helper_createAssetMetadata(t, ctx, dao, 1)
		assetID := assets[0].ID
		before := beforeAll[0]
		require.NotNil(t, before.VideoMetadata)
		require.NotNil(t, before.AudioMetadata)

		// Change fields
		vmNew := &models.VideoMetadata{
			DurationSec: 150,
			Container:   "mp4",
			MIMEType:    "video/mp4",
			SizeBytes:   1234567,
			OverallBPS:  3_000_000,
			VideoCodec:  "h265",
			Width:       1920,
			Height:      1080,
			FPSNum:      60,
			FPSDen:      1,
		}

		time.Sleep(2 * time.Millisecond)

		err := dao.UpdateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID:       assetID,
			VideoMetadata: vmNew,
			AudioMetadata: nil,
		})
		require.NoError(t, err)

		after, err := dao.GetAssetMetadata(ctx, assetID)
		require.NoError(t, err)
		require.NotNil(t, after)

		// Video changed
		require.NotNil(t, after.VideoMetadata)
		require.Equal(t, 150, after.VideoMetadata.DurationSec)
		require.Equal(t, "mp4", after.VideoMetadata.Container)
		require.Equal(t, "video/mp4", after.VideoMetadata.MIMEType)
		require.Equal(t, int64(1234567), after.VideoMetadata.SizeBytes)
		require.Equal(t, 3_000_000, after.VideoMetadata.OverallBPS)
		require.Equal(t, "h265", after.VideoMetadata.VideoCodec)
		require.Equal(t, 1920, after.VideoMetadata.Width)
		require.Equal(t, 1080, after.VideoMetadata.Height)
		require.Equal(t, 60, after.VideoMetadata.FPSNum)
		require.Equal(t, 1, after.VideoMetadata.FPSDen)
		require.False(t, after.VideoMetadata.UpdatedAt.Equal(before.VideoMetadata.UpdatedAt))

		// Audio untouched
		require.NotNil(t, after.AudioMetadata)
		require.Equal(t, before.AudioMetadata.Language, after.AudioMetadata.Language)
		require.Equal(t, before.AudioMetadata.Codec, after.AudioMetadata.Codec)
		require.Equal(t, before.AudioMetadata.Profile, after.AudioMetadata.Profile)
		require.Equal(t, before.AudioMetadata.Channels, after.AudioMetadata.Channels)
		require.Equal(t, before.AudioMetadata.ChannelLayout, after.AudioMetadata.ChannelLayout)
		require.Equal(t, before.AudioMetadata.SampleRate, after.AudioMetadata.SampleRate)
		require.Equal(t, before.AudioMetadata.BitRate, after.AudioMetadata.BitRate)
		require.True(t, after.AudioMetadata.UpdatedAt.Equal(before.AudioMetadata.UpdatedAt))
	})

	t.Run("audio update", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, beforeAll := helper_createAssetMetadata(t, ctx, dao, 1)
		assetID := assets[0].ID
		before := beforeAll[0]
		require.NotNil(t, before.VideoMetadata)
		require.NotNil(t, before.AudioMetadata)

		amNew := &models.AudioMetadata{
			Language:      "eng",
			Codec:         "eac3",
			Profile:       "DD+",
			Channels:      6,
			ChannelLayout: "5.1",
			SampleRate:    48000,
			BitRate:       768_000,
		}

		time.Sleep(2 * time.Millisecond)

		err := dao.UpdateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID:       assetID,
			VideoMetadata: nil,
			AudioMetadata: amNew,
		})
		require.NoError(t, err)

		after, err := dao.GetAssetMetadata(ctx, assetID)
		require.NoError(t, err)
		require.NotNil(t, after)

		// Audoi changed
		require.NotNil(t, after.AudioMetadata)
		require.Equal(t, "eng", after.AudioMetadata.Language)
		require.Equal(t, "eac3", after.AudioMetadata.Codec)
		require.Equal(t, "DD+", after.AudioMetadata.Profile)
		require.Equal(t, 6, after.AudioMetadata.Channels)
		require.Equal(t, "5.1", after.AudioMetadata.ChannelLayout)
		require.Equal(t, 48000, after.AudioMetadata.SampleRate)
		require.Equal(t, 768_000, after.AudioMetadata.BitRate)
		require.False(t, after.AudioMetadata.UpdatedAt.Equal(before.AudioMetadata.UpdatedAt))

		// Video untouched
		require.NotNil(t, after.VideoMetadata)
		require.Equal(t, before.VideoMetadata.DurationSec, after.VideoMetadata.DurationSec)
		require.Equal(t, before.VideoMetadata.Container, after.VideoMetadata.Container)
		require.Equal(t, before.VideoMetadata.MIMEType, after.VideoMetadata.MIMEType)
		require.Equal(t, before.VideoMetadata.SizeBytes, after.VideoMetadata.SizeBytes)
		require.Equal(t, before.VideoMetadata.OverallBPS, after.VideoMetadata.OverallBPS)
		require.Equal(t, before.VideoMetadata.VideoCodec, after.VideoMetadata.VideoCodec)
		require.Equal(t, before.VideoMetadata.Width, after.VideoMetadata.Width)
		require.Equal(t, before.VideoMetadata.Height, after.VideoMetadata.Height)
		require.Equal(t, before.VideoMetadata.FPSNum, after.VideoMetadata.FPSNum)
		require.Equal(t, before.VideoMetadata.FPSDen, after.VideoMetadata.FPSDen)
		require.True(t, after.VideoMetadata.UpdatedAt.Equal(before.VideoMetadata.UpdatedAt))
	})

	t.Run("video and audio update", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, beforeAll := helper_createAssetMetadata(t, ctx, dao, 1)
		assetID := assets[0].ID
		before := beforeAll[0]
		require.NotNil(t, before.VideoMetadata)
		require.NotNil(t, before.AudioMetadata)

		vmNew := &models.VideoMetadata{
			DurationSec: 200,
			Container:   "mp4",
			MIMEType:    "video/mp4",
			SizeBytes:   2222222,
			OverallBPS:  4_200_000,
			VideoCodec:  "h265",
			Width:       2560,
			Height:      1440,
			FPSNum:      30,
			FPSDen:      1,
		}
		amNew := &models.AudioMetadata{
			Language:      "eng",
			Codec:         "aac",
			Profile:       "LC",
			Channels:      2,
			ChannelLayout: "stereo",
			SampleRate:    44100,
			BitRate:       192_000,
		}

		time.Sleep(2 * time.Millisecond)

		err := dao.UpdateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID:       assetID,
			VideoMetadata: vmNew,
			AudioMetadata: amNew,
		})
		require.NoError(t, err)

		after, err := dao.GetAssetMetadata(ctx, assetID)
		require.NoError(t, err)
		require.NotNil(t, after)

		// Video changed
		require.NotNil(t, after.VideoMetadata)
		require.Equal(t, 200, after.VideoMetadata.DurationSec)
		require.Equal(t, int64(2222222), after.VideoMetadata.SizeBytes)
		require.Equal(t, 4_200_000, after.VideoMetadata.OverallBPS)
		require.Equal(t, "h265", after.VideoMetadata.VideoCodec)
		require.Equal(t, 2560, after.VideoMetadata.Width)
		require.Equal(t, 1440, after.VideoMetadata.Height)
		require.False(t, after.VideoMetadata.UpdatedAt.Equal(before.VideoMetadata.UpdatedAt))

		// Audio changed
		require.NotNil(t, after.AudioMetadata)
		require.Equal(t, "eng", after.AudioMetadata.Language)
		require.Equal(t, "aac", after.AudioMetadata.Codec)
		require.Equal(t, "LC", after.AudioMetadata.Profile)
		require.Equal(t, 2, after.AudioMetadata.Channels)
		require.Equal(t, "stereo", after.AudioMetadata.ChannelLayout)
		require.Equal(t, 44100, after.AudioMetadata.SampleRate)
		require.Equal(t, 192_000, after.AudioMetadata.BitRate)
		require.False(t, after.AudioMetadata.UpdatedAt.Equal(before.AudioMetadata.UpdatedAt))
	})

	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		require.ErrorIs(t, dao.UpdateAssetMetadata(ctx, nil), utils.ErrNilPtr)

		err := dao.UpdateAssetMetadata(ctx, &models.AssetMetadata{
			AssetID:       "",
			VideoMetadata: &models.VideoMetadata{DurationSec: 1},
		})
		require.ErrorIs(t, err, utils.ErrId)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteAssetMetadataByAssetIDs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 2)

		// Validate first and second asset metadata exists
		record, err := dao.GetAssetMetadata(ctx, assets[0].ID)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.NotNil(t, record.VideoMetadata)
		require.NotNil(t, record.AudioMetadata)

		record, err = dao.GetAssetMetadata(ctx, assets[1].ID)
		require.NoError(t, err)
		require.NotNil(t, record)
		require.NotNil(t, record.VideoMetadata)
		require.NotNil(t, record.AudioMetadata)

		require.NoError(t, dao.DeleteAssetMetadataByAssetIDs(ctx, assets[0].ID, assets[1].ID))

		record, err = dao.GetAssetMetadata(ctx, assets[0].ID)
		require.NoError(t, err)
		require.Nil(t, record)

		record, err = dao.GetAssetMetadata(ctx, assets[1].ID)
		require.NoError(t, err)
		require.Nil(t, record)
	})

	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 1)
		asset := assets[0]

		require.NoError(t, dao.DeleteAssetMetadataByAssetIDs(ctx, "does-not-exist"))

		record, err := dao.GetAssetMetadata(ctx, asset.ID)
		require.NoError(t, err)
		require.NotNil(t, record)
	})

	t.Run("No IDs", func(t *testing.T) {
		dao, ctx := setup(t)

		err := dao.DeleteAssetMetadataByAssetIDs(ctx)
		require.NoError(t, err)
	})

	t.Run("Cascade", func(t *testing.T) {
		dao, ctx := setup(t)

		assets, _ := helper_createAssetMetadata(t, ctx, dao, 1)
		asset := assets[0]

		dbOpts := NewOptions().WithWhere(squirrel.Eq{"id": asset.ID})
		require.NoError(t, dao.DeleteAssets(ctx, dbOpts))

		record, err := dao.GetAssetMetadata(ctx, asset.ID)
		require.NoError(t, err)
		require.Nil(t, record)
	})
}
