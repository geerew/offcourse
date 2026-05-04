package dao

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/queryparser"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateLesson inserts a new lesson record
func (dao *DAO) CreateLesson(ctx context.Context, lesson *models.Lesson) error {
	if err := lessonValidation(lesson); err != nil {
		return err
	}

	if lesson.ID == "" {
		lesson.RefreshId()
	}

	lesson.RefreshCreatedAt()
	lesson.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.LESSON_TABLE).
		WithData(map[string]interface{}{
			models.BASE_ID:          lesson.ID,
			models.LESSON_COURSE_ID: lesson.CourseID,
			models.LESSON_TITLE:     lesson.Title,
			models.LESSON_PREFIX:    lesson.Prefix,
			models.LESSON_MODULE:    lesson.Module,
			models.BASE_CREATED_AT:  lesson.CreatedAt,
			models.BASE_UPDATED_AT:  lesson.UpdatedAt,
		})

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetLesson gets a record from the lessons table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
//
// Performs a minimum of 3 db queries (lesson, attachments, assets) with the potential to make 2
// additional db queries (asset progress and asset metadata)
//
// Asset progress is not included by default. It can be enabled by calling `WithUserProgress()`
// on the options. This will add an additional db query
//
// Asset metadata is not included by default. It can be enabled by calling `WithAssetMetadata()`
// on the options. This will add an additional db query
//
// Note: Something can definitely be done to reduce the number of db queries whether via
// JOINS, parallelisation, or something else
func (dao *DAO) GetLesson(ctx context.Context, dbOpts *Options) (*models.Lesson, error) {
	builderOpts := newBuilderOptions(models.LESSON_TABLE).
		WithColumns(models.LessonColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	lesson, err := getGeneric[models.Lesson](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if lesson == nil {
		return nil, nil
	}

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress
	includeMetadata := dbOpts != nil && dbOpts.IncludeAssetMetadata

	if err := attachLessonRelations(ctx, dao, []*models.Lesson{lesson}, includeProgress, includeMetadata); err != nil {
		return nil, err
	}

	return lesson, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListLessons gets all records from the lessons table based upon the where clause and pagination
// in the options
//
// Performs a minimum of 3 db queries (lesson, attachments, assets) with the potential to make 2
// additional db queries (asset progress and asset metadata)
//
// Asset progress is not included by default. It can be enabled by calling `WithUserProgress()`
// on the options. This will add an additional db query
//
// Asset metadata is not included by default. It can be enabled by calling `WithAssetMetadata()`
// on the options. This will add an additional db query
//
// Note: Something can definitely be done to reduce the number of db queries whether via
// JOINS, parallelisation, or something else
func (dao *DAO) ListLessons(ctx context.Context, dbOpts *Options) ([]*models.Lesson, error) {
	if err := parseLessonApiQuery(dbOpts); err != nil {
		return nil, err
	}

	// Fetch lessons
	builderOpts := newBuilderOptions(models.LESSON_TABLE).
		WithColumns(models.LessonColumns()...).
		SetDbOpts(dbOpts)

	lessons, err := listGeneric[models.Lesson](ctx, dao, *builderOpts)
	if err != nil || len(lessons) == 0 {
		return lessons, err
	}

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress
	includeMetadata := dbOpts != nil && dbOpts.IncludeAssetMetadata

	if err := attachLessonRelations(ctx, dao, lessons, includeProgress, includeMetadata); err != nil {
		return nil, err
	}

	return lessons, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UpdateLesson updates a single lesson record
func (dao *DAO) UpdateLesson(ctx context.Context, lesson *models.Lesson) error {
	if err := lessonValidation(lesson); err != nil {
		return err
	}

	if lesson.ID == "" {
		return utils.ErrId
	}

	lesson.RefreshUpdatedAt()

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: lesson.ID})

	builderOpts := newBuilderOptions(models.LESSON_TABLE).
		WithData(map[string]interface{}{
			models.LESSON_TITLE:    lesson.Title,
			models.LESSON_PREFIX:   lesson.Prefix,
			models.LESSON_MODULE:   lesson.Module,
			models.BASE_UPDATED_AT: lesson.UpdatedAt,
		}).
		SetDbOpts(dbOpts)

	_, err := updateGeneric(ctx, dao, *builderOpts)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteLessons deletes records from the LESSONs table
//
// Errors when a WHERE clause is not provided.
func (dao *DAO) DeleteLessons(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.LESSON_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// lessonValidation validates the lesson fields
func lessonValidation(ag *models.Lesson) error {
	if ag == nil {
		return utils.ErrNilPtr
	}

	if ag.CourseID == "" {
		return utils.ErrCourseId
	}

	if ag.Title == "" {
		return utils.ErrTitle
	}

	if !ag.Prefix.Valid || ag.Prefix.Int16 < 0 {
		return utils.ErrPrefix
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var defaultLessonsListOrderBy = []string{
	models.LESSON_TABLE_MODULE + " asc",
	models.LESSON_TABLE_PREFIX + " asc",
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseLessonApiQuery parses dbOpts.ApiQuery and applies sort only (no WHERE from `q`).
func parseLessonApiQuery(dbOpts *Options) error {
	if dbOpts == nil {
		return nil
	}

	q := dbOpts.ApiQuery

	if q == "" {
		if len(dbOpts.OrderBy) == 0 && dbOpts.OrderByClause == nil {
			dbOpts.WithOrderBy(defaultLessonsListOrderBy...)
		}

		return nil
	}

	parsed, err := queryparser.Parse(q, nil)
	if err != nil {
		return fmt.Errorf("%w: %w", utils.ErrApiQueryParse, err)
	}

	if parsed == nil {
		dbOpts.WithOrderBy(defaultLessonsListOrderBy...)
		return nil
	}

	if len(parsed.Sort) > 0 {
		dbOpts.WithOrderBy(parsed.Sort...)
	} else {
		dbOpts.WithOrderBy(defaultLessonsListOrderBy...)
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// attachLessonRelations attaches attachments and assets to the given lessons
//
// It will also optionally attach asset progress and metadata to the assets
func attachLessonRelations(ctx context.Context, dao *DAO, lessons []*models.Lesson, includeProgress, includeMetadata bool) error {
	if len(lessons) == 0 {
		return nil
	}

	lessonIDs := utils.Map(lessons, func(l *models.Lesson) string { return l.ID })

	// Attachments (ordered by title)
	dbOpts := NewOptions().
		WithWhere(squirrel.Eq{models.ATTACHMENT_LESSON_ID: lessonIDs}).
		WithOrderBy(
			models.ATTACHMENT_TABLE_LESSON_ID+" ASC",
			models.ATTACHMENT_TABLE_TITLE+" ASC",
		)

	attachmentRecords, err := dao.ListAttachments(ctx, dbOpts)
	if err != nil {
		return err
	}

	attMap := make(map[string][]*models.Attachment)
	for _, record := range attachmentRecords {
		attMap[record.LessonID] = append(attMap[record.LessonID], record)
	}

	// Assets (ordered by prefix + sub_prefix)
	dbOpts = NewOptions().
		WithWhere(squirrel.Eq{models.ASSET_LESSON_ID: lessonIDs}).
		WithOrderBy(
			models.ASSET_TABLE_LESSON_ID+" ASC",
			models.ASSET_TABLE_PREFIX+" ASC",
			models.ASSET_TABLE_SUB_PREFIX+" ASC",
		)
	if includeProgress {
		dbOpts.WithUserProgress()
	}
	if includeMetadata {
		dbOpts.WithAssetMetadata()
	}

	assetRecords, err := dao.ListAssets(ctx, dbOpts)
	if err != nil {
		return err
	}

	assetMap := make(map[string][]*models.Asset)
	for _, record := range assetRecords {
		assetMap[record.LessonID] = append(assetMap[record.LessonID], record)
	}

	// Attach attachments and assets to the lessons
	for _, lesson := range lessons {
		lesson.Attachments = attMap[lesson.ID]
		lesson.Assets = assetMap[lesson.ID]
	}

	return nil
}
