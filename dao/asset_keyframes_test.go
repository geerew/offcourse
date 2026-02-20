package dao

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_CreateAssetKeyframes(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test asset first
	asset := createTestAsset(t, ctx, dao, "test-asset-1")

	t.Run("success", func(t *testing.T) {
		keyframes := &models.AssetKeyframes{
			AssetID:        asset.ID,
			KeyframesSlice: []float64{0.0, 2.5, 5.0, 7.5, 10.0},
			IsComplete:     true,
		}

		err := dao.CreateAssetKeyframes(ctx, keyframes)
		require.NoError(t, err)
		assert.NotEmpty(t, keyframes.ID)
		assert.NotEmpty(t, keyframes.CreatedAt)
		assert.NotEmpty(t, keyframes.UpdatedAt)
	})

	t.Run("nil keyframes", func(t *testing.T) {
		err := dao.CreateAssetKeyframes(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil pointer")
	})

	t.Run("empty asset ID", func(t *testing.T) {
		keyframes := &models.AssetKeyframes{
			AssetID: "",
		}

		err := dao.CreateAssetKeyframes(ctx, keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset id cannot be empty")
	})

	t.Run("invalid keyframes", func(t *testing.T) {
		keyframes := &models.AssetKeyframes{
			AssetID:        "test-asset-2",
			KeyframesSlice: []float64{5.0, 2.5, 10.0}, // Not ascending
		}

		err := dao.CreateAssetKeyframes(ctx, keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in ascending order")
	})

	t.Run("duplicate asset ID", func(t *testing.T) {
		// Create a test asset first
		asset := createTestAsset(t, ctx, dao, "test-asset-3")

		// First create
		keyframes1 := &models.AssetKeyframes{
			AssetID:        asset.ID,
			KeyframesSlice: []float64{0.0, 2.5},
			IsComplete:     false,
		}

		err := dao.CreateAssetKeyframes(ctx, keyframes1)
		require.NoError(t, err)

		// Try to create duplicate
		keyframes2 := &models.AssetKeyframes{
			AssetID:        asset.ID,
			KeyframesSlice: []float64{0.0, 2.5, 5.0},
			IsComplete:     true,
		}

		err = dao.CreateAssetKeyframes(ctx, keyframes2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "UNIQUE constraint failed")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_GetAssetKeyframes(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test asset first
	asset := createTestAsset(t, ctx, dao, "test-asset-4")

	// Setup test data
	keyframes := &models.AssetKeyframes{
		AssetID:        asset.ID,
		KeyframesSlice: []float64{0.0, 2.5, 5.0, 7.5, 10.0},
		IsComplete:     true,
	}

	err := dao.CreateAssetKeyframes(ctx, keyframes)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		retrieved, err := dao.GetAssetKeyframes(ctx, asset.ID)
		require.NoError(t, err)
		require.NotNil(t, retrieved)

		assert.Equal(t, keyframes.ID, retrieved.ID)
		assert.Equal(t, asset.ID, retrieved.AssetID)
		assert.Equal(t, []float64{0.0, 2.5, 5.0, 7.5, 10.0}, retrieved.KeyframesSlice)
		assert.True(t, retrieved.IsComplete)
		assert.NotEmpty(t, retrieved.CreatedAt)
		assert.NotEmpty(t, retrieved.UpdatedAt)
	})

	t.Run("not found", func(t *testing.T) {
		retrieved, err := dao.GetAssetKeyframes(ctx, "non-existent-asset")
		require.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("empty asset ID", func(t *testing.T) {
		retrieved, err := dao.GetAssetKeyframes(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset id cannot be empty")
		assert.Nil(t, retrieved)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_UpdateAssetKeyframes(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test asset first
	asset := createTestAsset(t, ctx, dao, "test-asset-5")

	// Setup test data
	keyframes := &models.AssetKeyframes{
		AssetID:        asset.ID,
		KeyframesSlice: []float64{0.0, 2.5},
		IsComplete:     false,
	}

	err := dao.CreateAssetKeyframes(ctx, keyframes)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		keyframes.KeyframesSlice = []float64{0.0, 2.5, 5.0, 7.5, 10.0}
		keyframes.IsComplete = true

		err := dao.UpdateAssetKeyframes(ctx, keyframes)
		require.NoError(t, err)

		// Verify update
		retrieved, err := dao.GetAssetKeyframes(ctx, asset.ID)
		require.NoError(t, err)
		assert.Equal(t, []float64{0.0, 2.5, 5.0, 7.5, 10.0}, retrieved.KeyframesSlice)
		assert.True(t, retrieved.IsComplete)
	})

	t.Run("nil keyframes", func(t *testing.T) {
		err := dao.UpdateAssetKeyframes(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil pointer")
	})

	t.Run("empty ID", func(t *testing.T) {
		keyframes := &models.AssetKeyframes{
			AssetID: "test-asset-6",
		}

		err := dao.UpdateAssetKeyframes(ctx, keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id cannot be empty")
	})

	t.Run("invalid keyframes", func(t *testing.T) {
		keyframes := &models.AssetKeyframes{
			AssetID:        "test-asset-7",
			KeyframesSlice: []float64{5.0, 2.5, 10.0}, // Not ascending
		}
		keyframes.RefreshId()

		err := dao.UpdateAssetKeyframes(ctx, keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not in ascending order")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_DeleteAssetKeyframes(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test asset first
	asset := createTestAsset(t, ctx, dao, "test-asset-8")

	// Setup test data
	keyframes := &models.AssetKeyframes{
		AssetID:        asset.ID,
		KeyframesSlice: []float64{0.0, 2.5, 5.0},
		IsComplete:     true,
	}

	err := dao.CreateAssetKeyframes(ctx, keyframes)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		err := dao.DeleteAssetKeyframes(ctx, asset.ID)
		require.NoError(t, err)

		// Verify deletion
		retrieved, err := dao.GetAssetKeyframes(ctx, asset.ID)
		require.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("empty asset ID", func(t *testing.T) {
		err := dao.DeleteAssetKeyframes(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset id cannot be empty")
	})

	t.Run("not found", func(t *testing.T) {
		err := dao.DeleteAssetKeyframes(ctx, "non-existent-asset")
		require.NoError(t, err) // Delete should not error if not found
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_DeleteAssetKeyframesById(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test asset first
	asset := createTestAsset(t, ctx, dao, "test-asset-9")

	// Setup test data
	keyframes := &models.AssetKeyframes{
		AssetID:        asset.ID,
		KeyframesSlice: []float64{0.0, 2.5, 5.0},
		IsComplete:     true,
	}

	err := dao.CreateAssetKeyframes(ctx, keyframes)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		err := dao.DeleteAssetKeyframesById(ctx, keyframes.ID)
		require.NoError(t, err)

		// Verify deletion
		retrieved, err := dao.GetAssetKeyframes(ctx, asset.ID)
		require.Error(t, err)
		assert.Nil(t, retrieved)
	})

	t.Run("empty ID", func(t *testing.T) {
		err := dao.DeleteAssetKeyframesById(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "id cannot be empty")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_ListAssetKeyframes(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	// Create test assets first
	asset1 := createTestAsset(t, ctx, dao, "test-asset-10")
	asset2 := createTestAsset(t, ctx, dao, "test-asset-11")

	// Setup test data
	keyframes1 := &models.AssetKeyframes{
		AssetID:        asset1.ID,
		KeyframesSlice: []float64{0.0, 2.5},
		IsComplete:     false,
	}

	keyframes2 := &models.AssetKeyframes{
		AssetID:        asset2.ID,
		KeyframesSlice: []float64{0.0, 2.5, 5.0, 7.5},
		IsComplete:     true,
	}

	err := dao.CreateAssetKeyframes(ctx, keyframes1)
	require.NoError(t, err)

	err = dao.CreateAssetKeyframes(ctx, keyframes2)
	require.NoError(t, err)

	t.Run("list all", func(t *testing.T) {
		list, err := dao.ListAssetKeyframes(ctx, nil)
		require.NoError(t, err)
		assert.Len(t, list, 2)
	})

	t.Run("with where clause", func(t *testing.T) {
		opts := &Options{
			Where: squirrel.Eq{models.ASSET_KEYFRAMES_IS_COMPLETE: true},
		}

		list, err := dao.ListAssetKeyframes(ctx, opts)
		require.NoError(t, err)
		assert.Len(t, list, 1)
		assert.Equal(t, asset2.ID, list[0].AssetID)
	})

	t.Run("with limit", func(t *testing.T) {
		p := pagination.New(1, 1)
		opts := &Options{
			Pagination: p,
		}

		list, err := dao.ListAssetKeyframes(ctx, opts)
		require.NoError(t, err)
		assert.Len(t, list, 1)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_ExistsAssetKeyframes(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	// Create a test asset first
	asset := createTestAsset(t, ctx, dao, "test-asset-12")

	// Setup test data
	keyframes := &models.AssetKeyframes{
		AssetID:        asset.ID,
		KeyframesSlice: []float64{0.0, 2.5, 5.0},
		IsComplete:     true,
	}

	err := dao.CreateAssetKeyframes(ctx, keyframes)
	require.NoError(t, err)

	t.Run("exists", func(t *testing.T) {
		exists, err := dao.ExistsAssetKeyframes(ctx, asset.ID)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("not exists", func(t *testing.T) {
		exists, err := dao.ExistsAssetKeyframes(ctx, "non-existent-asset")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("empty asset ID", func(t *testing.T) {
		exists, err := dao.ExistsAssetKeyframes(ctx, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset id cannot be empty")
		assert.False(t, exists)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_GetAssetKeyframesCount(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("empty database", func(t *testing.T) {
		count, err := dao.GetAssetKeyframesCount(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	// Create test assets first
	asset1 := createTestAsset(t, ctx, dao, "test-asset-13")
	asset2 := createTestAsset(t, ctx, dao, "test-asset-14")

	// Setup test data
	keyframes1 := &models.AssetKeyframes{
		AssetID:        asset1.ID,
		KeyframesSlice: []float64{0.0, 2.5},
		IsComplete:     false,
	}

	keyframes2 := &models.AssetKeyframes{
		AssetID:        asset2.ID,
		KeyframesSlice: []float64{0.0, 2.5, 5.0},
		IsComplete:     true,
	}

	err := dao.CreateAssetKeyframes(ctx, keyframes1)
	require.NoError(t, err)

	err = dao.CreateAssetKeyframes(ctx, keyframes2)
	require.NoError(t, err)

	t.Run("total count", func(t *testing.T) {
		count, err := dao.GetAssetKeyframesCount(ctx, nil)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("filtered count", func(t *testing.T) {
		opts := &Options{
			Where: squirrel.Eq{models.ASSET_KEYFRAMES_IS_COMPLETE: true},
		}

		count, err := dao.GetAssetKeyframesCount(ctx, opts)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestDAO_UpsertAssetKeyframes(t *testing.T) {
	dao, cleanup := setupTestDAO(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("create new", func(t *testing.T) {
		// Create a test asset first
		asset := createTestAsset(t, ctx, dao, "test-asset-15")

		keyframes := &models.AssetKeyframes{
			AssetID:        asset.ID,
			KeyframesSlice: []float64{0.0, 2.5, 5.0},
			IsComplete:     true,
		}

		err := dao.UpsertAssetKeyframes(ctx, keyframes)
		require.NoError(t, err)
		assert.NotEmpty(t, keyframes.ID)

		// Verify creation
		retrieved, err := dao.GetAssetKeyframes(ctx, asset.ID)
		require.NoError(t, err)
		assert.Equal(t, []float64{0.0, 2.5, 5.0}, retrieved.KeyframesSlice)
		assert.True(t, retrieved.IsComplete)
	})

	t.Run("update existing", func(t *testing.T) {
		// Create a test asset first
		asset := createTestAsset(t, ctx, dao, "test-asset-16")

		// First create
		keyframes := &models.AssetKeyframes{
			AssetID:        asset.ID,
			KeyframesSlice: []float64{0.0, 2.5},
			IsComplete:     false,
		}

		err := dao.UpsertAssetKeyframes(ctx, keyframes)
		require.NoError(t, err)
		originalID := keyframes.ID

		// Now update
		keyframes.KeyframesSlice = []float64{0.0, 2.5, 5.0, 7.5}
		keyframes.IsComplete = true

		err = dao.UpsertAssetKeyframes(ctx, keyframes)
		require.NoError(t, err)
		assert.Equal(t, originalID, keyframes.ID) // ID should be preserved

		// Verify update
		retrieved, err := dao.GetAssetKeyframes(ctx, asset.ID)
		require.NoError(t, err)
		assert.Equal(t, []float64{0.0, 2.5, 5.0, 7.5}, retrieved.KeyframesSlice)
		assert.True(t, retrieved.IsComplete)
	})

	t.Run("nil keyframes", func(t *testing.T) {
		err := dao.UpsertAssetKeyframes(ctx, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil pointer")
	})

	t.Run("empty asset ID", func(t *testing.T) {
		keyframes := &models.AssetKeyframes{
			AssetID: "",
		}

		err := dao.UpsertAssetKeyframes(ctx, keyframes)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "asset id cannot be empty")
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// setupTestDAO creates a test DAO with in-memory database
func setupTestDAO(t *testing.T) (*DAO, func()) {
	appFs := appfs.New(afero.NewMemMapFs())

	dbManager, err := database.NewSQLiteManager(&database.DatabaseManagerConfig{
		DataDir: "./oc_data",
		AppFs:   appFs,
		Testing: true,
	})
	require.NoError(t, err)

	dao := &DAO{
		db: dbManager.DataDb,
	}

	// Create required parent records for foreign key constraints
	ctx := context.Background()

	// Create a course
	course := &models.Course{
		Title: "Test Course",
		Path:  "/test/course",
	}
	course.RefreshId()
	course.RefreshCreatedAt()
	course.RefreshUpdatedAt()

	_, err = dao.db.ExecContext(ctx, `
		INSERT INTO courses (id, title, path, available, duration, initial_scan, maintenance, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, course.ID, course.Title, course.Path, false, 0, false, false, course.CreatedAt, course.UpdatedAt)
	require.NoError(t, err)

	// Create a lesson
	lesson := &models.Lesson{
		CourseID: course.ID,
		Title:    "Test Lesson",
		Prefix:   sql.NullInt16{Int16: 1, Valid: true},
	}
	lesson.RefreshId()
	lesson.RefreshCreatedAt()
	lesson.RefreshUpdatedAt()

	_, err = dao.db.ExecContext(ctx, `
		INSERT INTO lessons (id, course_id, title, prefix, module, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, lesson.ID, lesson.CourseID, lesson.Title, lesson.Prefix, nil, lesson.CreatedAt, lesson.UpdatedAt)
	require.NoError(t, err)

	// Create a user for foreign key constraints
	user := &models.User{
		Username:     "testuser",
		DisplayName:  "Test User",
		PasswordHash: "hashedpassword",
		Role:         "user",
	}
	user.RefreshId()
	user.RefreshCreatedAt()
	user.RefreshUpdatedAt()

	_, err = dao.db.ExecContext(ctx, `
		INSERT INTO users (id, username, display_name, password_hash, role, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Username, user.DisplayName, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt)
	require.NoError(t, err)

	cleanup := func() {
		// No explicit cleanup needed for in-memory database
	}

	return dao, cleanup
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// createTestAsset creates a test asset for foreign key constraints
func createTestAsset(t *testing.T, ctx context.Context, dao *DAO, title string) *models.Asset {
	// Get the first course and lesson from the database
	var courseID, lessonID string
	err := dao.db.QueryRowContext(ctx, "SELECT id FROM courses LIMIT 1").Scan(&courseID)
	require.NoError(t, err)

	err = dao.db.QueryRowContext(ctx, "SELECT id FROM lessons LIMIT 1").Scan(&lessonID)
	require.NoError(t, err)

	assetType := types.AssetVideo

	asset := &models.Asset{
		CourseID: courseID,
		LessonID: lessonID,
		Title:    title,
		Path:     "/test/path/" + title,
		Hash:     "hash-" + title,
		Weight:   1,
		Prefix:   sql.NullInt16{Int16: 1, Valid: true},
		Type:     assetType,
	}
	asset.RefreshId()
	asset.RefreshCreatedAt()
	asset.RefreshUpdatedAt()

	_, err = dao.db.ExecContext(ctx, `
		INSERT INTO assets (id, course_id, lesson_id, title, path, hash, weight, prefix, type, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, asset.ID, asset.CourseID, asset.LessonID, asset.Title, asset.Path, asset.Hash, asset.Weight, asset.Prefix, asset.Type, asset.CreatedAt, asset.UpdatedAt)
	require.NoError(t, err)

	return asset
}
