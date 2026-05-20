package cron

import (
	"context"
	"fmt"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/appfs"
	"github.com/geerew/off-course/utils/mocks"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestCourseAvailability_Run(t *testing.T) {
	t.Run("update", func(t *testing.T) {
		testApp, ctx := setup(t)

		appDao := dao.New(testApp.DbManager.DataDb)

		courses := []*models.Course{}
		for i := range 3 {
			course := &models.Course{Title: fmt.Sprintf("course %d", i), Path: fmt.Sprintf("/course-%d", i), Available: false}
			require.NoError(t, appDao.CreateCourse(ctx, course))
			courses = append(courses, course)
		}

		ca := &courseAvailability{
			db:        testApp.DbManager.DataDb,
			dao:       appDao,
			appFs:     testApp.AppFs,
			logger:    testApp.Logger.WithCron(),
			batchSize: 2,
		}

		err := ca.run()
		require.NoError(t, err)

		for _, course := range courses {
			require.Nil(t, testApp.AppFs.Fs.MkdirAll(course.Path, 0755))
		}

		err = ca.run()
		require.NoError(t, err)

		for _, course := range courses {
			dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
			course, err := ca.dao.GetCourse(ctx, dbOpts)
			require.NoError(t, err)
			require.True(t, course.Available)
		}
	})

	t.Run("stat error", func(t *testing.T) {
		testApp, ctx := setup(t)

		appDao := dao.New(testApp.DbManager.DataDb)

		course := &models.Course{Title: "course 1", Path: "/course-1", Available: false}
		require.NoError(t, appDao.CreateCourse(ctx, course))

		fsWithError := &mocks.MockFsWithError{
			Fs:          afero.NewMemMapFs(),
			ErrToReturn: fmt.Errorf("stat error"),
		}

		ca := &courseAvailability{
			db:        testApp.DbManager.DataDb,
			dao:       appDao,
			appFs:     appfs.New(fsWithError),
			logger:    testApp.Logger.WithCron(),
			batchSize: 1,
		}

		err := ca.run()
		require.Equal(t, fmt.Errorf("stat error"), err)
	})

	t.Run("db error", func(t *testing.T) {
		testApp, _ := setup(t)

		db := testApp.DbManager.DataDb
		_, err := db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TABLE)
		require.NoError(t, err)

		ca := &courseAvailability{
			db:        db,
			dao:       dao.New(db),
			appFs:     testApp.AppFs,
			logger:    testApp.Logger.WithCron(),
			batchSize: 1,
		}

		err = ca.run()
		require.ErrorContains(t, err, "no such table: "+models.COURSE_TABLE)
	})
}
