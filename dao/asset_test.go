package dao

import (
	"context"
	"database/sql"
	"fmt"
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

func Test_CreateAsset(t *testing.T) {
	// Test successfully inserting an asset record
	t.Run("success", func(t *testing.T) {
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
			Path:     "/course-1/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)
		require.ErrorIs(t, dao.CreateAsset(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to invalid fields
	t.Run("invalid", func(t *testing.T) {
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

		asset := &models.Asset{}
		require.ErrorIs(t, dao.CreateAsset(ctx, asset), utils.ErrCourseId)

		asset.CourseID = course.ID
		require.ErrorIs(t, dao.CreateAsset(ctx, asset), utils.ErrLessonId)

		asset.LessonID = lesson.ID
		require.ErrorIs(t, dao.CreateAsset(ctx, asset), utils.ErrTitle)

		asset.Title = "Asset 1"
		require.ErrorIs(t, dao.CreateAsset(ctx, asset), utils.ErrPrefix)

		asset.Prefix = sql.NullInt16{Int16: 1, Valid: true}
		require.ErrorIs(t, dao.CreateAsset(ctx, asset), utils.ErrPath)
	})

}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetAsset(t *testing.T) {
	// Test successfully retrieving an asset record
	t.Run("success", func(t *testing.T) {
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
			Path:     "/course-1/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: asset.ID})
		record, err := dao.GetAsset(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, asset.ID, record.ID)
		require.Nil(t, record.Progress)
	})

	// Test successfully retrieving an asset record with relations
	t.Run("success with relations", func(t *testing.T) {
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
			Path:     "/course-1/01 asset.mp4",
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

		// With progress
		dbOpts := NewOptions().
			WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: asset.ID}).
			WithUserProgress()

		record, err := dao.GetAsset(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, asset.ID, record.ID)
		require.Nil(t, record.Progress)
		require.Nil(t, record.AssetMetadata)

		// Set progress
		assetProgress := &models.AssetProgress{
			AssetID:     asset.ID,
			Position:    100,
			Completed:   true,
			CompletedAt: types.NowDateTime(),
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		// Get the asset again with progress
		record, err = dao.GetAsset(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, asset.ID, record.ID)
		require.NotNil(t, record.Progress)
		require.Equal(t, 100, record.Progress.Position)
		require.True(t, record.Progress.Completed)
		require.False(t, record.Progress.CompletedAt.IsZero())
		require.Nil(t, record.AssetMetadata)

		// Get the asset with metadata
		dbOpts.WithAssetMetadata()

		record, err = dao.GetAsset(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, asset.ID, record.ID)
		require.NotNil(t, record.Progress)
		require.NotNil(t, record.AssetMetadata)
		require.NotNil(t, record.AssetMetadata.VideoMetadata)
		require.NotNil(t, record.AssetMetadata.VideoMetadata)
		require.Equal(t, 120, record.AssetMetadata.VideoMetadata.DurationSec)
		require.Equal(t, 1280, record.AssetMetadata.VideoMetadata.Width)
		require.Nil(t, record.AssetMetadata.AudioMetadata)

		// Create another user
		user2 := &models.User{
			Username:     "user2",
			DisplayName:  "User 2",
			PasswordHash: "hash",
			Role:         types.UserRoleUser,
		}
		require.NoError(t, dao.CreateUser(ctx, user2))

		// Set the principal to user2, which is picked up when interacting with progress
		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		principal.UserID = user2.ID
		ctx = context.WithValue(ctx, types.PrincipalContextKey, principal)

		// Create an asset progress (and therefore another course progress) for the
		// new user
		assetProgress2 := &models.AssetProgress{
			AssetID: asset.ID,

			Position: 200,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress2))

		// Confirm there are 2 asset progress records
		builderOpts := newBuilderOptions(models.ASSET_PROGRESS_TABLE)
		count, err := countGeneric(ctx, dao, *builderOpts)
		require.NoError(t, err)
		require.Equal(t, 2, count)

		// Get the course for user 2
		record, err = dao.GetAsset(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, asset.ID, record.ID)
		require.NotNil(t, record.Progress)
		require.Equal(t, 200, record.Progress.Position)
		require.False(t, record.Progress.Completed)
		require.True(t, record.Progress.CompletedAt.IsZero())
		require.NotNil(t, record.AssetMetadata)
		require.NotNil(t, record.AssetMetadata.VideoMetadata)
		require.NotNil(t, record.AssetMetadata.VideoMetadata)
		require.Equal(t, 120, record.AssetMetadata.VideoMetadata.DurationSec)
		require.Equal(t, 1280, record.AssetMetadata.VideoMetadata.Width)
		require.Nil(t, record.AssetMetadata.AudioMetadata)
	})

	// Test successfully querying without a where clause
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		record, err := dao.GetAsset(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})

	// Test error due to missing principal
	t.Run("missing principal", func(t *testing.T) {
		dao, _ := setup(t)

		dbOpts := NewOptions().WithUserProgress()
		record, err := dao.GetAsset(context.Background(), dbOpts)
		require.ErrorIs(t, err, utils.ErrPrincipal)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListAssets(t *testing.T) {
	// Test successfully retrieving all asset records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		assets := []*models.Asset{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
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
				Path:     fmt.Sprintf("/course-%d/01 asset.mp4", i),
			}
			require.NoError(t, dao.CreateAsset(ctx, asset))
			assets = append(assets, asset)

			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListAssets(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, assets[i].ID, record.ID)
			require.Nil(t, record.Progress)
		}
	})

	// Test successfully retrieving all asset records with relations
	t.Run("success with relations", func(t *testing.T) {
		dao, ctx := setup(t)

		assets := []*models.Asset{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
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
				Path:     fmt.Sprintf("/course-%d/01 asset.mp4", i),
			}
			require.NoError(t, dao.CreateAsset(ctx, asset))
			assets = append(assets, asset)

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

			time.Sleep(1 * time.Millisecond)
		}

		// List with progress
		dbOpts := NewOptions().WithUserProgress()

		records, err := dao.ListAssets(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		// Ensure no progress when none exists
		for i, record := range records {
			require.Equal(t, assets[i].ID, record.ID)
			require.Nil(t, record.Progress)
			require.Nil(t, record.AssetMetadata)
		}

		// Generate progress for the default user
		assetProgress := &models.AssetProgress{
			AssetID: assets[0].ID,

			Position:  20,
			Completed: true,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress))

		// List again)
		records, err = dao.ListAssets(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		// Ensure first asset has progress, others do not
		for i, record := range records {
			require.Equal(t, assets[i].ID, record.ID)
			if i == 0 {
				require.NotNil(t, record.Progress)
				require.Equal(t, 20, record.Progress.Position)
				require.True(t, record.Progress.Completed)
				require.False(t, record.Progress.CompletedAt.IsZero())
			} else {
				require.Nil(t, record.Progress)
			}
		}

		// Create another user
		user2 := &models.User{
			Username:     "user2",
			DisplayName:  "User 2",
			PasswordHash: "hash",
			Role:         types.UserRoleUser,
		}
		require.NoError(t, dao.CreateUser(ctx, user2))

		// Set the principal to user2, which is picked up when interacting with progress
		principal := ctx.Value(types.PrincipalContextKey).(types.Principal)
		principal.UserID = user2.ID
		ctx = context.WithValue(ctx, types.PrincipalContextKey, principal)

		// For course 2, create an asset progress (and therefore another course progress) for the
		// new user
		assetProgress2 := &models.AssetProgress{
			AssetID:   assets[1].ID,
			Position:  50,
			Completed: true,
		}
		require.NoError(t, dao.UpsertAssetProgress(ctx, assetProgress2))

		// List again
		records, err = dao.ListAssets(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		// Ensure second asset has progress for user2, others do not
		for i, record := range records {
			require.Equal(t, assets[i].ID, record.ID)
			if i == 1 {
				require.NotNil(t, record.Progress)
				require.Equal(t, 50, record.Progress.Position)
				require.True(t, record.Progress.Completed)
				require.False(t, record.Progress.CompletedAt.IsZero())
			} else {
				require.Nil(t, record.Progress)
			}
		}

		// List with progress and metadata
		dbOpts.WithAssetMetadata()

		records, err = dao.ListAssets(ctx, dbOpts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		// Ensure second asset has progress and metadata for user2, others have metadata only
		for i, record := range records {
			require.Equal(t, assets[i].ID, record.ID)
			require.NotNil(t, record.AssetMetadata)
			require.NotNil(t, record.AssetMetadata.VideoMetadata)
			require.Equal(t, 120, record.AssetMetadata.VideoMetadata.DurationSec)
			require.Equal(t, 1280, record.AssetMetadata.VideoMetadata.Width)
			require.Nil(t, record.AssetMetadata.AudioMetadata)

			if i == 1 {
				require.NotNil(t, record.Progress)
				require.Equal(t, 50, record.Progress.Position)
				require.True(t, record.Progress.Completed)
				require.False(t, record.Progress.CompletedAt.IsZero())
			} else {
				require.Nil(t, record.Progress)
			}
		}
	})

	// Test successfully retrieving no asset records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		records, err := dao.ListAssets(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered asset records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		assets := []*models.Asset{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
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
				Path:     fmt.Sprintf("/course-%d/01 asset.mp4", i),
			}
			require.NoError(t, dao.CreateAsset(ctx, asset))
			assets = append(assets, asset)
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.ASSET_TABLE_CREATED_AT + " DESC")
		records, err := dao.ListAssets(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, assets[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.ASSET_TABLE_CREATED_AT + " ASC")
		records, err = dao.ListAssets(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, assets[i].ID, record.ID)
		}
	})

	// Test successfully retrieving selected asset records
	t.Run("where", func(t *testing.T) {
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
			Path:     fmt.Sprintf("/course-%d/01 asset.mp4", 1),
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: asset.ID})
		records, err := dao.ListAssets(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, asset.ID, records[0].ID)
	})

	// Test successfully retrieving paginated asset records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		assets := []*models.Asset{}
		for i := range 17 {
			course := &models.Course{Title: fmt.Sprintf("Course %d", i), Path: fmt.Sprintf("/course-%d", i)}
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
				Path:     fmt.Sprintf("/course-%d/01 asset.mp4", i),
			}
			require.NoError(t, dao.CreateAsset(ctx, asset))
			assets = append(assets, asset)
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().WithPagination(pagination.New(1, 10))
		records, err := dao.ListAssets(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, assets[0].ID, records[0].ID)
		require.Equal(t, assets[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().WithPagination(pagination.New(2, 10))
		records, err = dao.ListAssets(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, assets[10].ID, records[0].ID)
		require.Equal(t, assets[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpdateAsset(t *testing.T) {
	// Test successfully updating an asset record
	t.Run("success", func(t *testing.T) {
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

		originalAsset := &models.Asset{
			CourseID: course.ID,
			LessonID: lesson.ID,
			Title:    "Asset 1",
			Prefix:   sql.NullInt16{Int16: 1, Valid: true},
			Module:   "Module 1",
			Type:     types.MustAsset("mp4"),
			Path:     "/course-1/01 asset.mp4",
			FileSize: 1024,
			ModTime:  time.Now().GoString(),
			Hash:     "abc123",
		}
		require.NoError(t, dao.CreateAsset(ctx, originalAsset))

		time.Sleep(1 * time.Millisecond)

		newLesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group 2",
			Prefix:   sql.NullInt16{Int16: 2, Valid: true},
			Module:   "Module 2",
		}
		require.NoError(t, dao.CreateLesson(ctx, newLesson))

		updatedAsset := &models.Asset{
			Base:     originalAsset.Base,
			CourseID: "54321",
			LessonID: newLesson.ID,
			Title:    "Updated Asset",
			Prefix:   sql.NullInt16{Int16: 2, Valid: true},
			Module:   "Updated Module",
			Type:     types.MustAsset("mkv"),
			Path:     "/course-1/02 asset.mkv",
			FileSize: 2048,
			ModTime:  time.Now().Add(1 * time.Hour).GoString(),
			Hash:     "def456",
		}
		require.NoError(t, dao.UpdateAsset(ctx, updatedAsset))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: originalAsset.ID})
		record, err := dao.GetAsset(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, originalAsset.ID, record.ID)                    // No change
		require.Equal(t, originalAsset.CourseID, record.CourseID)        // No change
		require.True(t, record.CreatedAt.Equal(originalAsset.CreatedAt)) // No change
		require.Equal(t, updatedAsset.Title, record.Title)               // Changed
		require.Equal(t, updatedAsset.Path, record.Path)                 // Changed
		require.Equal(t, updatedAsset.LessonID, record.LessonID)         // Changed
		require.Equal(t, updatedAsset.Prefix, record.Prefix)             // Changed
		require.Equal(t, updatedAsset.Module, record.Module)             // Changed
		require.Equal(t, updatedAsset.Type, record.Type)                 // Changed
		require.Equal(t, updatedAsset.FileSize, record.FileSize)         // Changed
		require.NotEqual(t, originalAsset.ModTime, record.ModTime)       // Changed
		require.Equal(t, updatedAsset.Hash, record.Hash)                 // Changed
		require.NotEqual(t, originalAsset.UpdatedAt, record.UpdatedAt)   // Changed

	})

	// Test error due to invalid fields
	t.Run("invalid", func(t *testing.T) {
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

		asset := &models.Asset{}

		// Course ID
		require.ErrorIs(t, dao.UpdateAsset(ctx, asset), utils.ErrCourseId)
		asset.CourseID = course.ID

		// Lesson ID
		require.ErrorIs(t, dao.UpdateAsset(ctx, asset), utils.ErrLessonId)
		asset.LessonID = lesson.ID

		// Title
		require.ErrorIs(t, dao.UpdateAsset(ctx, asset), utils.ErrTitle)
		asset.Title = "Asset 1"

		// Prefix
		require.ErrorIs(t, dao.UpdateAsset(ctx, asset), utils.ErrPrefix)
		asset.Prefix = sql.NullInt16{Int16: 1, Valid: true}

		// Path
		require.ErrorIs(t, dao.UpdateAsset(ctx, asset), utils.ErrPath)
		asset.Path = "/course-1/01 asset.mp4"

		// ID
		require.ErrorIs(t, dao.UpdateAsset(ctx, asset), utils.ErrId)

		// Nil Model
		require.ErrorIs(t, dao.UpdateAsset(ctx, nil), utils.ErrNilPtr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteAsset(t *testing.T) {
	// Test successfully deleting an asset record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group",
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
			Path:     "/course/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: asset.ID})
		require.Nil(t, dao.DeleteAssets(ctx, opts))

		records, err := dao.ListAssets(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent asset record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group",
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
			Path:     "/course/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		opts := NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteAssets(ctx, opts))

		records, err := dao.ListAssets(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, asset.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group",
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
			Path:     "/course/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		require.ErrorIs(t, dao.DeleteAssets(ctx, nil), utils.ErrWhere)

		records, err := dao.ListAssets(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, asset.ID, records[0].ID)
	})

	// Test cascading delete of asset records when deleting a course
	t.Run("cascade", func(t *testing.T) {
		dao, ctx := setup(t)

		course := &models.Course{Title: "Course", Path: "/course"}
		require.NoError(t, dao.CreateCourse(ctx, course))

		lesson := &models.Lesson{
			CourseID: course.ID,
			Title:    "Asset Group",
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
			Path:     "/course/01 asset.mp4",
		}
		require.NoError(t, dao.CreateAsset(ctx, asset))

		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		require.Nil(t, dao.DeleteCourses(ctx, dbOpts))

		records, err := dao.ListAssets(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, records)
	})
}
