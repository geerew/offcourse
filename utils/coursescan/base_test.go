package coursescan

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/filesystem"
	"github.com/geerew/off-course/utils/cardcache"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/media"
	"github.com/geerew/off-course/utils/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func setup(t *testing.T) (*CourseScan, context.Context) {
	t.Helper()

	testLogger := logger.NilLogger()
	fs := filesystem.New(afero.NewMemMapFs())
	dataDir := "./oc_data"

	dbManager, err := database.NewSQLiteManager(&database.DatabaseManagerConfig{
		DataDir: dataDir,
		FS:   fs,
		Testing: true,
	})
	require.NoError(t, err)
	require.NotNil(t, dbManager)

	tools, err := media.NewTools()
	if err != nil {
		t.Skip("ffmpeg/ffprobe not available; skipping test")
	}

	cardCache, err := cardcache.New(&cardcache.CardCacheConfig{
		CachePath: dataDir,
		FS:   fs,
		Logger:    testLogger,
	})
	require.NoError(t, err)

	courseScan := New(&CourseScanConfig{
		Db:        dbManager.DataDb,
		FS:   fs,
		Logger:    testLogger,
		Tools:     tools,
		CardCache: cardCache,
	})

	user := &models.User{
		Username:     "test-user",
		DisplayName:  "Test User",
		PasswordHash: "test-password",
		Role:         types.UserRoleAdmin,
	}
	require.NoError(t, courseScan.dao.CreateUser(context.Background(), user))

	principal := types.Principal{
		UserID: user.ID,
		Role:   user.Role,
	}
	ctx := context.WithValue(context.Background(), types.PrincipalContextKey, principal)

	return courseScan, ctx
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func intPtr(i int) *int {
	return &i
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestScanner_Add(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		scanner, ctx := setup(t)

		course1 := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course1))

		scan1, err := scanner.Add(ctx, course1.ID)
		require.NoError(t, err)
		require.Equal(t, course1.ID, scan1.CourseID)

		course2 := &models.Course{Title: "Course 2", Path: "/course-2"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course2))

		scan2, err := scanner.Add(ctx, course2.ID)
		require.NoError(t, err)
		require.Equal(t, course2.ID, scan2.CourseID)
	})

	t.Run("duplicate", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		first, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)
		require.Equal(t, course.ID, first.CourseID)

		// Add again
		second, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)
		require.Equal(t, second.ID, first.ID)
		// Note: Log assertions removed as we no longer have access to log entries in the new logger system
	})

	t.Run("invalid course", func(t *testing.T) {
		scanner, ctx := setup(t)

		scan, err := scanner.Add(ctx, "1234")
		require.ErrorIs(t, err, utils.ErrCourseNotFound)
		require.Nil(t, scan)
	})

	t.Run("concurrent add same course", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		// Add the same course concurrently from multiple goroutines
		const numGoroutines = 10
		results := make([]*ScanState, numGoroutines)
		errors := make([]error, numGoroutines)

		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx], errors[idx] = scanner.Add(ctx, course.ID)
			}(i)
		}
		wg.Wait()

		// All should succeed
		for i := 0; i < numGoroutines; i++ {
			require.NoError(t, errors[i], "goroutine %d should not error", i)
			require.NotNil(t, results[i], "goroutine %d should return a scan", i)
		}

		// All should return the same scan ID (no duplicates created)
		firstScanID := results[0].ID
		for i := 1; i < numGoroutines; i++ {
			require.Equal(t, firstScanID, results[i].ID, "all goroutines should return the same scan ID")
		}

		// Verify only one scan exists in the map
		allScans := scanner.GetAllScans()
		require.Len(t, allScans, 1, "only one scan should exist")
		require.Equal(t, course.ID, allScans[0].CourseID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestScanner_Worker(t *testing.T) {
	t.Run("jobs", func(t *testing.T) {
		scanner, ctx := setup(t)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course-%d", i)}
			require.NoError(t, scanner.dao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		go scanner.Worker(ctx, func(context.Context, *CourseScan, *ScanState) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		// Add the courses
		for i := range 3 {
			scan, err := scanner.Add(ctx, courses[i].ID)
			require.NoError(t, err)
			require.Equal(t, scan.CourseID, courses[i].ID)
		}

		// Poll until all scans are processed
		require.Eventually(t, func() bool {
			return len(scanner.GetAllScans()) == 0
		}, 2*time.Second, 50*time.Millisecond, "Scans should be processed and removed")

		// Add the first 2 courses (again)
		for i := range 2 {
			scan, err := scanner.Add(ctx, courses[i].ID)
			require.NoError(t, err)
			require.Equal(t, scan.CourseID, courses[i].ID)
		}

		// Poll until all scans are processed again
		require.Eventually(t, func() bool {
			return len(scanner.GetAllScans()) == 0
		}, 2*time.Second, 50*time.Millisecond, "Scans should be processed and removed")
	})

	t.Run("error processing", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		go scanner.Worker(ctx, func(context.Context, *CourseScan, *ScanState) error {
			time.Sleep(1 * time.Millisecond)
			return errors.New("processing error")
		})

		scan, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)
		require.Equal(t, scan.CourseID, course.ID)

		// Poll until scan is processed (even if it errors, it should be removed)
		require.Eventually(t, func() bool {
			return len(scanner.GetAllScans()) == 0
		}, 2*time.Second, 50*time.Millisecond, "Scan should be processed and removed even on error")
	})
}
