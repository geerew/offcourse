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

func Test_CreateAssetMetadata(t *testing.T) {
	// Test successfully inserting an asset metadata record (video only)
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

	// Test successfully inserting an asset metadata record (video + audio)
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

	// Test successfully inserting an asset metadata record (no metadata)
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

	// Test error due to nil pointer
	t.Run("nil metadata", func(t *testing.T) {
		dao, ctx := setup(t)
		require.ErrorIs(t, dao.CreateAssetMetadata(ctx, nil), utils.ErrNilPtr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetAssetMetadata(t *testing.T) {
	// Test successfully retrieving an asset metadata record
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

	// Test querying a non-existent asset metadata record
	t.Run("asset id not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetAssetMetadata(ctx, "does-not-exist")
		require.NoError(t, err)
		require.Nil(t, record)

	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteAssetMetadata(t *testing.T) {
	// Test cascading delete of asset metadata records
	t.Run("cascade", func(t *testing.T) {
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
