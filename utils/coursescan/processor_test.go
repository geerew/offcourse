package coursescan

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/coursemetadata"
	"github.com/geerew/off-course/utils/types"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestScanner_Processor(t *testing.T) {
	t.Run("scan nil", func(t *testing.T) {
		scanner, ctx := setup(t)

		err := Processor(ctx, scanner, nil)
		require.ErrorIs(t, err, ErrNilScan)
	})

	t.Run("error getting course", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		_, err = scanner.db.ExecContext(context.Background(), "DROP TABLE IF EXISTS "+models.COURSE_TABLE)
		require.NoError(t, err)

		err = Processor(ctx, scanner, scanState)
		require.ErrorContains(t, err, fmt.Sprintf("no such table: %s", models.COURSE_TABLE))
	})

	t.Run("course unavailable", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: true}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		err = Processor(ctx, scanner, scanState)
		require.NoError(t, err)

		// Note: Log assertions removed as we no longer have access to log entries in the new logger system
	})

	t.Run("mark course available", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Available: false}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)

		err = Processor(ctx, scanner, scanState)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		record, err := scanner.dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.True(t, record.Available)
	})

	t.Run("card", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})

		// Add card at the root
		{
			scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)
			scanner.appFs.Fs.Create(filepath.Join(course.Path, "card.jpg"))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			record, err := scanner.dao.GetCourse(ctx, dbOpts)
			require.NoError(t, err)
			require.Equal(t, filepath.Join(course.Path, "card.jpg"), record.CardPath)
		}

		// Ignore card in chapter
		{
			scanner.appFs.Fs.Remove(filepath.Join(course.Path, "card.jpg"))
			scanner.appFs.Fs.Create(filepath.Join(course.Path, "01 Chapter 1", "card.jpg"))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			record, err := scanner.dao.GetCourse(ctx, dbOpts)
			require.NoError(t, err)
			require.Empty(t, record.CardPath)
		}

		// Ignore additional cards at the root
		{
			scanner.appFs.Fs.Create(filepath.Join(course.Path, "card.jpg"))
			scanner.appFs.Fs.Create(filepath.Join(course.Path, "card.png"))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			record, err := scanner.dao.GetCourse(ctx, dbOpts)
			require.NoError(t, err)
			require.Equal(t, filepath.Join(course.Path, "card.jpg"), record.CardPath)
		}
	})

	t.Run("re-optimizes card when content changes", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		cardPath := filepath.Join(course.Path, "card.jpg")
		require.NoError(t, scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm))
		require.NoError(t, afero.WriteFile(scanner.appFs.Fs, cardPath, []byte("card-content-v1"), os.ModePerm))

		scanState := NewScanState(course.ID, course.Path, course.Title)

		changed, err := handleCourseCard(ctx, scanner, course, cardPath, course.ID, course.Path, scanState)
		require.NoError(t, err)
		require.True(t, changed)
		initialHash := course.CardHash
		require.NotEmpty(t, initialHash)

		optimizedPath, err := scanner.cardCache.GetCardPath(course.ID)
		require.NoError(t, err)
		require.NoError(t, afero.WriteFile(scanner.appFs.Fs, optimizedPath, []byte("cached-webp"), os.ModePerm))

		require.NoError(t, afero.WriteFile(scanner.appFs.Fs, cardPath, []byte("card-content-v2-different"), os.ModePerm))

		changed, err = handleCourseCard(ctx, scanner, course, cardPath, course.ID, course.Path, scanState)
		require.NoError(t, err)
		require.True(t, changed)
		require.NotEqual(t, initialHash, course.CardHash)
	})

	t.Run("ignore files", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/file 1", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/file.file", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/file.avi", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/ - file.avi", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/- - file.avi", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/.avi", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/-1 - file.avi", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/a - file.avi", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/1.1 - file.avi", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/2.3-file.avi", course.Path))
		scanner.appFs.Fs.Create(fmt.Sprintf("%s/1file.avi", course.Path))

		err = Processor(ctx, scanner, scanState)
		require.NoError(t, err)

		count, err := scanner.dao.CountAssets(ctx, nil)
		require.NoError(t, err)
		require.Zero(t, count)
	})

	t.Run("assets", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().
			WithWhere(squirrel.Eq{models.LESSON_TABLE_COURSE_ID: course.ID}).
			WithOrderBy(models.LESSON_TABLE_MODULE+" asc", models.LESSON_TABLE_PREFIX+" asc")

		var lessons []*models.Lesson

		// Add file 1, file 2 (create op)
		{
			scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 file 1.mkv", course.Path), []byte("hash 1"), os.ModePerm)
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/02 file 2.pdf", course.Path), []byte("hash 2"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 2)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, "file 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsVideo())
			require.Equal(t, "0657190350cbea662b6c15d703d9c7482308e511504d3308306d0f1ede153a34", lessons[0].Assets[0].Hash)

			require.Len(t, lessons[1].Assets, 1)
			require.Equal(t, "file 2", lessons[1].Assets[0].Title)
			require.Equal(t, course.ID, lessons[1].Assets[0].CourseID)
			require.Equal(t, 2, int(lessons[1].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[1].Assets[0].Module)
			require.True(t, lessons[1].Assets[0].Type.IsPDF())
			require.Equal(t, "ac4f5d7f5ca1f7b2a9e8107ca793b5ead43a1d04afdafabc9488e93b5d738b41", lessons[1].Assets[0].Hash)
		}

		// Add file 1 under a chapter (create op)
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 Chapter 1/01 file 1.pdf", course.Path), []byte("hash 3"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 3)

			require.Len(t, lessons[2].Assets, 1)
			require.Equal(t, "file 1", lessons[2].Assets[0].Title)
			require.Equal(t, course.ID, lessons[2].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[2].Assets[0].Prefix.Int16))
			require.Equal(t, "01 Chapter 1", lessons[2].Assets[0].Module)
			require.True(t, lessons[2].Assets[0].Type.IsPDF())
		}

		// Delete file 1 in chapter (delete op)
		{
			scanner.appFs.Fs.Remove(fmt.Sprintf("%s/01 Chapter 1/01 file 1.pdf", course.Path))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 2)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/01 file 1.mkv", course.Path), lessons[0].Assets[0].Path)
			require.Len(t, lessons[1].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/02 file 2.pdf", course.Path), lessons[1].Assets[0].Path)
		}

		// Rename file 2 to file 3 (update op)
		{
			existingAssetID := lessons[1].Assets[0].ID
			scanner.appFs.Fs.Rename(fmt.Sprintf("%s/02 file 2.pdf", course.Path), fmt.Sprintf("%s/03 file 3.pdf", course.Path))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 2)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/01 file 1.mkv", course.Path), lessons[0].Assets[0].Path)
			require.Len(t, lessons[1].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/03 file 3.pdf", course.Path), lessons[1].Assets[0].Path)

			require.Equal(t, existingAssetID, lessons[1].Assets[0].ID)
		}

		// Replace file 3 with new content (replace op)
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/03 file 3.pdf", course.Path), []byte("new hash 3"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 2)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/01 file 1.mkv", course.Path), lessons[0].Assets[0].Path)

			require.Len(t, lessons[1].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/03 file 3.pdf", course.Path), lessons[1].Assets[0].Path)
			require.Equal(t, "file 3", lessons[1].Assets[0].Title)
			require.Equal(t, course.ID, lessons[1].Assets[0].CourseID)
			require.Equal(t, 3, int(lessons[1].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[1].Assets[0].Module)
			require.True(t, lessons[1].Assets[0].Type.IsPDF())
			require.Equal(t, "638441f6644d2bf0a20dcf8f6e0b6b7df670c2ab14986af7b027796570fafbe0", lessons[1].Assets[0].Hash)
		}

		// Swap file 1 and file 3 (swap op)
		{
			scanner.appFs.Fs.Rename(fmt.Sprintf("%s/01 file 1.mkv", course.Path), fmt.Sprintf("%s/03 file 3.pdf.temp", course.Path))
			scanner.appFs.Fs.Rename(fmt.Sprintf("%s/03 file 3.pdf", course.Path), fmt.Sprintf("%s/01 file 1.mkv", course.Path))
			scanner.appFs.Fs.Rename(fmt.Sprintf("%s/03 file 3.pdf.temp", course.Path), fmt.Sprintf("%s/03 file 3.pdf", course.Path))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 2)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/01 file 1.mkv", course.Path), lessons[0].Assets[0].Path)
			require.Len(t, lessons[1].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/03 file 3.pdf", course.Path), lessons[1].Assets[0].Path)

			require.Equal(t, "638441f6644d2bf0a20dcf8f6e0b6b7df670c2ab14986af7b027796570fafbe0", lessons[0].Assets[0].Hash)
			require.Equal(t, "0657190350cbea662b6c15d703d9c7482308e511504d3308306d0f1ede153a34", lessons[1].Assets[0].Hash)
		}

		// Delete file 1 and move file 3 to file 1 (overwrite op)
		{
			require.NoError(t, scanner.appFs.Fs.Remove(fmt.Sprintf("%s/01 file 1.mkv", course.Path)))

			require.NoError(t, scanner.appFs.Fs.Rename(
				fmt.Sprintf("%s/03 file 3.pdf", course.Path),
				fmt.Sprintf("%s/01 file 1.mkv", course.Path),
			))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, fmt.Sprintf("%s/01 file 1.mkv", course.Path), lessons[0].Assets[0].Path)

			require.Equal(t, "0657190350cbea662b6c15d703d9c7482308e511504d3308306d0f1ede153a34", lessons[0].Assets[0].Hash)
		}

		// Delete all files but keep the course directory
		{
			// Delete all files but keep the course directory
			scanner.appFs.Fs.RemoveAll(fmt.Sprintf("%s/01 file 1.mkv", course.Path))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 0)
		}

		// Add file 1 with a sub-prefix (create op)
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 lesson 1 {01 video 1}.mkv", course.Path), []byte("hash 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)
			require.Len(t, lessons[0].Assets, 1)

			require.Equal(t, "lesson 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Equal(t, 1, int(lessons[0].Assets[0].SubPrefix.Int16))
			require.Equal(t, "video 1", lessons[0].Assets[0].SubTitle)
		}

		// Add file 2 with a sub-prefix (no op)
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 lesson 1 {02 video 2}.mkv", course.Path), []byte("hash 2"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err = scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)
			require.Len(t, lessons[0].Assets, 2)

			require.Equal(t, "lesson 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Equal(t, 1, int(lessons[0].Assets[0].SubPrefix.Int16))
			require.Equal(t, "video 1", lessons[0].Assets[0].SubTitle)

			require.Equal(t, "lesson 1", lessons[0].Assets[1].Title)
			require.Equal(t, course.ID, lessons[0].Assets[1].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[1].Prefix.Int16))
			require.Equal(t, 2, int(lessons[0].Assets[1].SubPrefix.Int16))
			require.Equal(t, "video 2", lessons[0].Assets[1].SubTitle)
		}
	})

	t.Run("attachments", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().
			WithOrderBy(models.LESSON_TABLE_MODULE+" asc", models.LESSON_TABLE_PREFIX+" asc").
			WithWhere(squirrel.Eq{models.LESSON_TABLE_COURSE_ID: course.ID})

		// Add asset (so the lesson is created)
		{
			scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 file 1.mkv", course.Path), []byte("hash 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, "file 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsVideo())
			require.Equal(t, "0657190350cbea662b6c15d703d9c7482308e511504d3308306d0f1ede153a34", lessons[0].Assets[0].Hash)
		}

		// Add attachment 1
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 attachment 1.url", course.Path), []byte("attachment 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Attachments, 1)
			require.Equal(t, "attachment 1.url", lessons[0].Attachments[0].Title)
			require.Equal(t, filepath.Join(course.Path, "01 attachment 1.url"), lessons[0].Attachments[0].Path)
		}

		// Add another attachment
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 attachment 2.url", course.Path), []byte("attachment 2"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Attachments, 2)
			require.Equal(t, "attachment 1.url", lessons[0].Attachments[0].Title)
			require.Equal(t, filepath.Join(course.Path, "01 attachment 1.url"), lessons[0].Attachments[0].Path)

			require.Equal(t, "attachment 2.url", lessons[0].Attachments[1].Title)
			require.Equal(t, filepath.Join(course.Path, "01 attachment 2.url"), lessons[0].Attachments[1].Path)
		}

		// Delete attachment
		{
			scanner.appFs.Fs.Remove(fmt.Sprintf("%s/01 attachment 1.url", course.Path))

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Attachments, 1)
			require.Equal(t, "attachment 2.url", lessons[0].Attachments[0].Title)
			require.Equal(t, filepath.Join(course.Path, "01 attachment 2.url"), lessons[0].Attachments[0].Path)
		}
	})

	t.Run("asset priority", func(t *testing.T) {
		scanner, ctx := setup(t)

		// Priority is VIDEO -> PDF -> MARKDOWN -> TEXT

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().
			WithOrderBy(models.LESSON_TABLE_MODULE+" asc", models.LESSON_TABLE_PREFIX+" asc").
			WithWhere(squirrel.Eq{models.LESSON_TABLE_COURSE_ID: course.ID})

		scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)

		// Add TEXT asset
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 text 1.txt", course.Path), []byte("text 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)
			require.Len(t, lessons[0].Assets, 1)
			require.Len(t, lessons[0].Attachments, 0)

			require.Equal(t, "text 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsText())
			require.Equal(t, "900a4469df00ccbfd0c145c6d1e4b7953dd0afafadd7534e3a4019e8d38fc663", lessons[0].Assets[0].Hash)
		}

		// Add MARKDOWN asset
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 markdown 1.md", course.Path), []byte("markdown 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, "markdown 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsMarkdown())
			require.Equal(t, "728cfbd456c4734229b7b545d69d182608eecc860c46081f51e3f1f108096eca", lessons[0].Assets[0].Hash)

			require.Len(t, lessons[0].Attachments, 1)
			require.Equal(t, filepath.Join(course.Path, "01 text 1.txt"), lessons[0].Attachments[0].Path)
		}

		// Add PDF asset
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 pdf 1.pdf", course.Path), []byte("pdf 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, "pdf 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsPDF())
			require.Equal(t, "9c9bfc90d1a2738f701a22c1ef10d42d5f2c285998a221eba9b7953e202bcf1a", lessons[0].Assets[0].Hash)

			require.Len(t, lessons[0].Attachments, 2)
			require.Equal(t, filepath.Join(course.Path, "01 markdown 1.md"), lessons[0].Attachments[0].Path)
			require.Equal(t, filepath.Join(course.Path, "01 text 1.txt"), lessons[0].Attachments[1].Path)
		}

		// Add VIDEO asset
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 video.mp4", course.Path), []byte("video"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, "video", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsVideo())
			require.Equal(t, "0cab1c9617404faf2b24e221e189ca5945813e14d3f766345b09ca13bbe28ffc", lessons[0].Assets[0].Hash)

			require.Len(t, lessons[0].Attachments, 3)
			require.Equal(t, filepath.Join(course.Path, "01 markdown 1.md"), lessons[0].Attachments[0].Path)
			require.Equal(t, filepath.Join(course.Path, "01 pdf 1.pdf"), lessons[0].Attachments[1].Path)
			require.Equal(t, filepath.Join(course.Path, "01 text 1.txt"), lessons[0].Attachments[2].Path)
		}

		// Add another PDF asset
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 pdf 2.pdf", course.Path), []byte("pdf 2"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, "video", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsVideo())
			require.Equal(t, "0cab1c9617404faf2b24e221e189ca5945813e14d3f766345b09ca13bbe28ffc", lessons[0].Assets[0].Hash)

			require.Len(t, lessons[0].Attachments, 4)
			require.Equal(t, filepath.Join(course.Path, "01 markdown 1.md"), lessons[0].Attachments[0].Path)
			require.Equal(t, filepath.Join(course.Path, "01 pdf 1.pdf"), lessons[0].Attachments[1].Path)
			require.Equal(t, filepath.Join(course.Path, "01 pdf 2.pdf"), lessons[0].Attachments[2].Path)
			require.Equal(t, filepath.Join(course.Path, "01 text 1.txt"), lessons[0].Attachments[3].Path)
		}
	})

	t.Run("asset with sub-prefix and sub-title", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().
			WithOrderBy(models.LESSON_TABLE_MODULE+" asc", models.LESSON_TABLE_PREFIX+" asc").
			WithWhere(squirrel.Eq{models.LESSON_TABLE_COURSE_ID: course.ID})

		// Add video 1 asset with sub-prefix of 1 and sub-title "Part 1"
		{
			scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 Group 1 {1 Video 1}.mp4", course.Path), []byte("video 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, "Group 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Equal(t, 1, int(lessons[0].Assets[0].SubPrefix.Int16))
			require.Equal(t, "Video 1", lessons[0].Assets[0].SubTitle)
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsVideo())
			require.Equal(t, "3b857b8441d7c9e734535d6b82f69a34c6fcd63ed0ef989ff03808ecb29a2f1f", lessons[0].Assets[0].Hash)

			require.Len(t, lessons[0].Attachments, 0)
		}

		// Add video 2 asset with sub-prefix of 2 and sub-title "Part 2"
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 Group 1 {2 Video 2}.mp4", course.Path), []byte("video 2"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 2)
			require.Equal(t, "3b857b8441d7c9e734535d6b82f69a34c6fcd63ed0ef989ff03808ecb29a2f1f", lessons[0].Assets[0].Hash)

			require.Equal(t, "Group 1", lessons[0].Assets[1].Title)
			require.Equal(t, course.ID, lessons[0].Assets[1].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[1].Prefix.Int16))
			require.Equal(t, 2, int(lessons[0].Assets[1].SubPrefix.Int16))
			require.Equal(t, "Video 2", lessons[0].Assets[1].SubTitle)
			require.Empty(t, lessons[0].Assets[1].Module)
			require.True(t, lessons[0].Assets[1].Type.IsVideo())
			require.Equal(t, "614ef49d4a1ef39bc763b7c9665f6f30a0eea3ec5ec10e04b897bdad9b973f9c", lessons[0].Assets[1].Hash)

			require.Len(t, lessons[0].Attachments, 0)
		}

		// Add video 3 asset with sub-prefix of 3 and no sub-title
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 Group 1 {3}.mp4", course.Path), []byte("video 3"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 3)

			require.Equal(t, "3b857b8441d7c9e734535d6b82f69a34c6fcd63ed0ef989ff03808ecb29a2f1f", lessons[0].Assets[0].Hash)
			require.Equal(t, "614ef49d4a1ef39bc763b7c9665f6f30a0eea3ec5ec10e04b897bdad9b973f9c", lessons[0].Assets[1].Hash)

			require.Equal(t, "Group 1", lessons[0].Assets[2].Title)
			require.Equal(t, course.ID, lessons[0].Assets[2].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[2].Prefix.Int16))
			require.Equal(t, 3, int(lessons[0].Assets[2].SubPrefix.Int16))
			require.Empty(t, lessons[0].Assets[2].SubTitle)
			require.Empty(t, lessons[0].Assets[2].Module)
			require.True(t, lessons[0].Assets[2].Type.IsVideo())
			require.Equal(t, "36d9fa5c21ca58822f678a5e1cebbaefcbcff37894771089cc608e8fbe32121e", lessons[0].Assets[2].Hash)

			require.Len(t, lessons[0].Attachments, 0)
		}

		// Add video 4 with no sub-prefix and no sub-title
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 Group 1.mp4", course.Path), []byte("video 4"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 3)

			require.Equal(t, "3b857b8441d7c9e734535d6b82f69a34c6fcd63ed0ef989ff03808ecb29a2f1f", lessons[0].Assets[0].Hash)
			require.Equal(t, "614ef49d4a1ef39bc763b7c9665f6f30a0eea3ec5ec10e04b897bdad9b973f9c", lessons[0].Assets[1].Hash)
			require.Equal(t, "36d9fa5c21ca58822f678a5e1cebbaefcbcff37894771089cc608e8fbe32121e", lessons[0].Assets[2].Hash)

			require.Len(t, lessons[0].Attachments, 1)
			require.Equal(t, filepath.Join(course.Path, "01 Group 1.mp4"), lessons[0].Attachments[0].Path)
		}

		// Add attachment
		{
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 attachment 1.txt", course.Path), []byte("attachment 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 3)
			require.Equal(t, "3b857b8441d7c9e734535d6b82f69a34c6fcd63ed0ef989ff03808ecb29a2f1f", lessons[0].Assets[0].Hash)
			require.Equal(t, "614ef49d4a1ef39bc763b7c9665f6f30a0eea3ec5ec10e04b897bdad9b973f9c", lessons[0].Assets[1].Hash)
			require.Equal(t, "36d9fa5c21ca58822f678a5e1cebbaefcbcff37894771089cc608e8fbe32121e", lessons[0].Assets[2].Hash)

			require.Len(t, lessons[0].Attachments, 2)
			require.Equal(t, filepath.Join(course.Path, "01 Group 1.mp4"), lessons[0].Attachments[0].Path)
			require.Equal(t, filepath.Join(course.Path, "01 attachment 1.txt"), lessons[0].Attachments[1].Path)
		}
	})

	t.Run("asset description", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().
			WithOrderBy(models.LESSON_TABLE_MODULE+" asc", models.LESSON_TABLE_PREFIX+" asc").
			WithWhere(squirrel.Eq{models.LESSON_TABLE_COURSE_ID: course.ID})

		// Add video 1 asset
		{
			scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)
			afero.WriteFile(scanner.appFs.Fs, fmt.Sprintf("%s/01 video 1.mp4", course.Path), []byte("video 1"), os.ModePerm)

			err := Processor(ctx, scanner, scanState)
			require.NoError(t, err)

			lessons, err := scanner.dao.ListLessons(ctx, dbOpts)
			require.NoError(t, err)
			require.Len(t, lessons, 1)

			require.Len(t, lessons[0].Assets, 1)
			require.Equal(t, "video 1", lessons[0].Assets[0].Title)
			require.Equal(t, course.ID, lessons[0].Assets[0].CourseID)
			require.Equal(t, 1, int(lessons[0].Assets[0].Prefix.Int16))
			require.Empty(t, lessons[0].Assets[0].Module)
			require.True(t, lessons[0].Assets[0].Type.IsVideo())
			require.Equal(t, "3b857b8441d7c9e734535d6b82f69a34c6fcd63ed0ef989ff03808ecb29a2f1f", lessons[0].Assets[0].Hash)
		}
	})

	t.Run("course description from metadata", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)

		// Write metadata file with description
		metadataPath := filepath.Join(course.Path, coursemetadata.MetadataFileName)
		metadataJSON := `{
  "description": "A test course description",
  "tags": ["test"]
}`
		require.NoError(t, afero.WriteFile(scanner.appFs.Fs, metadataPath, []byte(metadataJSON), 0644))

		err = Processor(ctx, scanner, scanState)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		record, err := scanner.dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Equal(t, "A test course description", record.Description)
	})

	t.Run("course description empty when metadata missing", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Description: "Original description"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)

		// No metadata file

		err = Processor(ctx, scanner, scanState)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		record, err := scanner.dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Empty(t, record.Description)
	})

	t.Run("course description empty string in metadata", func(t *testing.T) {
		scanner, ctx := setup(t)

		course := &models.Course{Title: "Course 1", Path: "/course-1", Description: "Original description"}
		require.NoError(t, scanner.dao.CreateCourse(ctx, course))

		scanState, err := scanner.Add(ctx, course.ID)
		require.NoError(t, err)

		scanner.appFs.Fs.Mkdir(course.Path, os.ModePerm)

		// Write metadata file with empty description
		metadataPath := filepath.Join(course.Path, coursemetadata.MetadataFileName)
		metadataJSON := `{
  "description": "",
  "tags": []
}`
		require.NoError(t, afero.WriteFile(scanner.appFs.Fs, metadataPath, []byte(metadataJSON), 0644))

		err = Processor(ctx, scanner, scanState)
		require.NoError(t, err)

		dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: course.ID})
		record, err := scanner.dao.GetCourse(ctx, dbOpts)
		require.NoError(t, err)
		require.Empty(t, record.Description)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestScanner_ParseFilename(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var tests = []string{
			// No prefix
			"file",
			"file.file",
			"file.avi",
			" - file.avi",
			"- - file.avi",
			".avi",
			// Invalid prefix
			"-1 - file.avi",
			"a - file.avi",
			"1.1 - file.avi",
			"2.3-file.avi",
			"1file.avi",
		}

		for _, tt := range tests {
			parsed := parseFilename("xx", tt)
			require.Nil(t, parsed)
		}
	})

	t.Run("assets", func(t *testing.T) {
		var tests = []struct {
			in       string
			expected *parsedFile
		}{
			// Video (varied)
			{"0    file 0.avi", &parsedFile{Prefix: 0, Title: "file 0", SubPrefix: nil, SubTitle: "", Ext: "avi", AssetType: types.MustAsset("avi"), IsCard: false, Original: "0    file 0.avi", NormalizedPath: "0    file 0.avi"}},
			{"001 file 1.mp4", &parsedFile{Prefix: 1, Title: "file 1", SubPrefix: nil, SubTitle: "", Ext: "mp4", AssetType: types.MustAsset("mp4"), IsCard: false, Original: "001 file 1.mp4", NormalizedPath: "001 file 1.mp4"}},
			{"1-file.ogg", &parsedFile{Prefix: 1, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "ogg", AssetType: types.MustAsset("ogg"), IsCard: false, Original: "1-file.ogg", NormalizedPath: "1-file.ogg"}},
			{"2 - file.webm", &parsedFile{Prefix: 2, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "webm", AssetType: types.MustAsset("webm"), IsCard: false, Original: "2 - file.webm", NormalizedPath: "2 - file.webm"}},
			{"3 -file.m4a", &parsedFile{Prefix: 3, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "m4a", AssetType: types.MustAsset("m4a"), IsCard: false, Original: "3 -file.m4a", NormalizedPath: "3 -file.m4a"}},
			{"4- file.opus", &parsedFile{Prefix: 4, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "opus", AssetType: types.MustAsset("opus"), IsCard: false, Original: "4- file.opus", NormalizedPath: "4- file.opus"}},
			{"5000 --- file.wav", &parsedFile{Prefix: 5000, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "wav", AssetType: types.MustAsset("wav"), IsCard: false, Original: "5000 --- file.wav", NormalizedPath: "5000 --- file.wav"}},
			{"0100 file.mp3", &parsedFile{Prefix: 100, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "mp3", AssetType: types.MustAsset("mp3"), IsCard: false, Original: "0100 file.mp3", NormalizedPath: "0100 file.mp3"}},
			// PDF (including mixed case)
			{"1 - doc.pdf", &parsedFile{Prefix: 1, Title: "doc", SubPrefix: nil, SubTitle: "", Ext: "pdf", AssetType: types.MustAsset("pdf"), IsCard: false, Original: "1 - doc.pdf", NormalizedPath: "1 - doc.pdf"}},
			{"2 - REPORT.PDF", &parsedFile{Prefix: 2, Title: "REPORT", SubPrefix: nil, SubTitle: "", Ext: "pdf", AssetType: types.MustAsset("pdf"), IsCard: false, Original: "2 - REPORT.PDF", NormalizedPath: "2 - REPORT.PDF"}},
			// Markdown
			{"5 notes.md", &parsedFile{Prefix: 5, Title: "notes", SubPrefix: nil, SubTitle: "", Ext: "md", AssetType: types.MustAsset("md"), IsCard: false, Original: "5 notes.md", NormalizedPath: "5 notes.md"}},
			// Text
			{"6 readme.txt", &parsedFile{Prefix: 6, Title: "readme", SubPrefix: nil, SubTitle: "", Ext: "txt", AssetType: types.MustAsset("txt"), IsCard: false, Original: "6 readme.txt", NormalizedPath: "6 readme.txt"}},
			// With sub-prefix but no subtitle
			{"01 file 0 {1}.avi", &parsedFile{Prefix: 1, Title: "file 0", SubPrefix: intPtr(1), SubTitle: "", Ext: "avi", AssetType: types.MustAsset("avi"), IsCard: false, Original: "01 file 0 {1}.avi", NormalizedPath: "01 file 0 {1}.avi"}},
			// With sub-prefix and subtitle
			{"01 file 0 {1 Part 1}.avi", &parsedFile{Prefix: 1, Title: "file 0", SubPrefix: intPtr(1), SubTitle: "Part 1", Ext: "avi", AssetType: types.MustAsset("avi"), IsCard: false, Original: "01 file 0 {1 Part 1}.avi", NormalizedPath: "01 file 0 {1 Part 1}.avi"}},
			{"01 file 0 {2 -   Part 2}.mp4", &parsedFile{Prefix: 1, Title: "file 0", SubPrefix: intPtr(2), SubTitle: "Part 2", Ext: "mp4", AssetType: types.MustAsset("mp4"), IsCard: false, Original: "01 file 0 {2 -   Part 2}.mp4", NormalizedPath: "01 file 0 {2 -   Part 2}.mp4"}},
			{"01 file 0 {}.mp4", &parsedFile{Prefix: 1, Title: "file 0", SubPrefix: nil, SubTitle: "", Ext: "mp4", AssetType: types.MustAsset("mp4"), IsCard: false, Original: "01 file 0 {}.mp4", NormalizedPath: "01 file 0 {}.mp4"}},
			// Description-like filenames
			{"01 description.md", &parsedFile{Prefix: 1, Title: "description", SubPrefix: nil, SubTitle: "", Ext: "md", AssetType: types.MustAsset("md"), IsCard: false, Original: "01 description.md", NormalizedPath: "01 description.md"}},
			{"02 Description.TXT", &parsedFile{Prefix: 2, Title: "Description", SubPrefix: nil, SubTitle: "", Ext: "txt", AssetType: types.MustAsset("txt"), IsCard: false, Original: "02 Description.TXT", NormalizedPath: "02 Description.TXT"}},
		}

		for _, tt := range tests {
			parsed := parseFilename(tt.in, tt.in)
			require.Equal(t, tt.expected, parsed, fmt.Sprintf("error for [%s]", tt.in))

		}
	})

	t.Run("cards", func(t *testing.T) {
		var tests = []struct {
			in       string
			expected *parsedFile
		}{
			{"card.jpg", &parsedFile{Prefix: 0, Title: "card", SubPrefix: nil, SubTitle: "", Ext: "jpg", AssetType: types.AssetType(""), IsCard: true, Original: "card.jpg", NormalizedPath: "card.jpg"}},
			{"card.jpeg", &parsedFile{Prefix: 0, Title: "card", SubPrefix: nil, SubTitle: "", Ext: "jpeg", AssetType: types.AssetType(""), IsCard: true, Original: "card.jpeg", NormalizedPath: "card.jpeg"}},
			{"card.png", &parsedFile{Prefix: 0, Title: "card", SubPrefix: nil, SubTitle: "", Ext: "png", AssetType: types.AssetType(""), IsCard: true, Original: "card.png", NormalizedPath: "card.png"}},
		}

		for _, tt := range tests {
			parsed := parseFilename(tt.in, tt.in)
			require.Equal(t, tt.expected, parsed, fmt.Sprintf("error for [%s]", tt.in))
		}
	})

	t.Run("attachments", func(t *testing.T) {
		var tests = []struct {
			in       string
			expected *parsedFile
		}{
			// No title
			{"01", &parsedFile{Prefix: 1, Title: "", SubPrefix: nil, SubTitle: "", Ext: "", AssetType: types.AssetType(""), IsCard: false, Original: "01", NormalizedPath: "01"}},
			{"200.pdf", &parsedFile{Prefix: 200, Title: "", SubPrefix: nil, SubTitle: "", Ext: "pdf", AssetType: types.MustAsset("pdf"), IsCard: false, Original: "200.pdf", NormalizedPath: "200.pdf"}},
			{"1 -.txt", &parsedFile{Prefix: 1, Title: "", SubPrefix: nil, SubTitle: "", Ext: "txt", AssetType: types.MustAsset("txt"), IsCard: false, Original: "1 -.txt", NormalizedPath: "1 -.txt"}},
			{"1 - .txt", &parsedFile{Prefix: 1, Title: "", SubPrefix: nil, SubTitle: "", Ext: "txt", AssetType: types.MustAsset("txt"), IsCard: false, Original: "1 - .txt", NormalizedPath: "1 - .txt"}},
			{"1 .txt", &parsedFile{Prefix: 1, Title: "", SubPrefix: nil, SubTitle: "", Ext: "txt", AssetType: types.MustAsset("txt"), IsCard: false, Original: "1 .txt", NormalizedPath: "1 .txt"}},
			{"1     .pdf", &parsedFile{Prefix: 1, Title: "", SubPrefix: nil, SubTitle: "", Ext: "pdf", AssetType: types.MustAsset("pdf"), IsCard: false, Original: "1     .pdf", NormalizedPath: "1     .pdf"}},
			// No extension
			{"1 - file", &parsedFile{Prefix: 1, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "", AssetType: types.AssetType(""), IsCard: false, Original: "1 - file", NormalizedPath: "1 - file"}},
			{"2 file", &parsedFile{Prefix: 2, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "", AssetType: types.AssetType(""), IsCard: false, Original: "2 file", NormalizedPath: "2 file"}},
			{"3-file", &parsedFile{Prefix: 3, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "", AssetType: types.AssetType(""), IsCard: false, Original: "3-file", NormalizedPath: "3-file"}},
			{"4 file", &parsedFile{Prefix: 4, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "", AssetType: types.AssetType(""), IsCard: false, Original: "4 file", NormalizedPath: "4 file"}},
			{"6 --- file", &parsedFile{Prefix: 6, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "", AssetType: types.AssetType(""), IsCard: false, Original: "6 --- file", NormalizedPath: "6 --- file"}},
			// Non-asset extensions (will be empty since NewAsset returns error)
			{"1 - file.exe", &parsedFile{Prefix: 1, Title: "file", SubPrefix: nil, SubTitle: "", Ext: "exe", AssetType: types.AssetType(""), IsCard: false, Original: "1 - file.exe", NormalizedPath: "1 - file.exe"}},
		}

		for _, tt := range tests {
			parsed := parseFilename(tt.in, tt.in)
			require.Equal(t, tt.expected, parsed, fmt.Sprintf("error for [%s]", tt.in))
		}
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestScanner_CategorizeFile(t *testing.T) {
	var tests = []struct {
		in       string
		expected FileCategory
	}{
		{"file", Ignore},
		{"file.file", Ignore},
		{"file.avi", Ignore},
		{" - file.avi", Ignore},
		{"- - file.avi", Ignore},
		{".avi", Ignore},
		{"-1 - file.avi", Ignore},
		{"a - file.avi", Ignore},
		{"1.1 - file.avi", Ignore},
		{"2.3-file.avi", Ignore},
		{"1file.avi", Ignore},
		// Asset
		{"0    file 0.avi", Asset},
		{"001 file 1.mp4", Asset},
		{"1-file.ogg", Asset},
		{"2 - file.webm", Asset},
		{"3 -file.m4a", Asset},
		{"4- file.opus", Asset},
		{"5000 --- file.wav", Asset},
		{"0100 file.mp3", Asset},
		{"1 - doc.pdf", Asset},
		{"2 - REPORT.PDF", Asset},
		{"5 notes.md", Asset},
		{"6 readme.txt", Asset},
		{"01 file 0 {}.mp4", Asset},
		// GroupedAsset
		{"01 file 0 {1}.avi", GroupedAsset},
		{"01 file 0 {1 Part 1}.avi", GroupedAsset},
		{"01 file 0 {2 -   Part 2}.mp4", GroupedAsset},
		// Card
		{"card.jpg", Card},
		{"card.jpeg", Card},
		// Attachment
		{"01", Attachment},
		{"200.pdf", Attachment},
		{"1 -.txt", Attachment},
		{"1 - .txt", Attachment},
		{"1 .txt", Attachment},
		{"1     .pdf", Attachment},
		{"1 - file", Attachment},
		{"2 file", Attachment},
		{"3-file", Attachment},
		{"6 --- file", Attachment},
		{"1 - file.exe", Attachment},
	}

	for _, tt := range tests {
		parsed := parseFilename("xx", tt.in)
		category := categorizeFile(parsed)
		require.Equal(t, tt.expected, category, fmt.Sprintf("error for [%s]", tt.in))
	}
}
