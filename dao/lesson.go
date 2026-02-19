package dao

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
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
func (dao *DAO) GetLesson(ctx context.Context, dbOpts *Options) (*models.Lesson, error) {
	// Fetch lesson
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

	// Fetch attachments (ordered by title)
	attachmentOpts := NewOptions().
		WithWhere(squirrel.Eq{models.ATTACHMENT_LESSON_ID: lesson.ID}).
		WithOrderBy(models.ATTACHMENT_TABLE_TITLE + " ASC")

	attachments, err := dao.ListAttachments(ctx, attachmentOpts)
	if err != nil {
		return nil, err
	}
	lesson.Attachments = attachments

	// Fetch assets (ordered by prefix + sub_prefix)
	assetDbOpts := NewOptions().
		WithWhere(squirrel.Eq{models.ASSET_LESSON_ID: lesson.ID}).
		WithOrderBy(models.ASSET_TABLE_PREFIX + " ASC, " + models.ASSET_TABLE_SUB_PREFIX + " ASC")

	if dbOpts != nil {
		assetDbOpts.IncludeUserProgress = dbOpts.IncludeUserProgress
		assetDbOpts.IncludeAssetMetadata = dbOpts.IncludeAssetMetadata
	}

	assets, err := dao.ListAssets(ctx, assetDbOpts)
	if err != nil {
		return nil, err
	}
	lesson.Assets = assets

	return lesson, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListLessons gets all records from the lessons table based upon the where clause and pagination
// in the options
func (dao *DAO) ListLessons(ctx context.Context, dbOpts *Options) ([]*models.Lesson, error) {
	// Fetch lessons
	builderOpts := newBuilderOptions(models.LESSON_TABLE).
		WithColumns(models.LessonColumns()...).
		SetDbOpts(dbOpts)

	// Override order by
	builderOpts.DbOpts.WithOrderBy(
		models.LESSON_TABLE_PREFIX+" ASC ",
		models.LESSON_TABLE_MODULE+" ASC",
	)

	lessons, err := listGeneric[models.Lesson](ctx, dao, *builderOpts)
	if err != nil || len(lessons) == 0 {
		return lessons, err
	}

	// Gather IDs
	ids := make([]string, 0, len(lessons))
	for i := range lessons {
		ids = append(ids, lessons[i].ID)
	}

	// Fetch attachments for all lessons, ordering by title
	attachmentDbOpts := NewOptions().
		WithWhere(squirrel.Eq{models.ATTACHMENT_LESSON_ID: ids}).
		WithOrderBy(
			models.ATTACHMENT_TABLE_LESSON_ID+" ASC",
			models.ATTACHMENT_TABLE_TITLE+" ASC",
		)

	attachments, err := dao.ListAttachments(ctx, attachmentDbOpts)
	if err != nil {
		return nil, err
	}

	attMap := make(map[string][]*models.Attachment)
	for _, a := range attachments {
		attMap[a.LessonID] = append(attMap[a.LessonID], a)
	}

	// Fetch assets for all lessons
	assetDbOpts := NewOptions().
		WithWhere(squirrel.Eq{models.ASSET_LESSON_ID: ids}).
		WithOrderBy(
			models.ASSET_TABLE_LESSON_ID+" ASC ",
			models.ASSET_TABLE_PREFIX+" ASC ",
			models.ASSET_TABLE_SUB_PREFIX+" ASC",
		)

	if dbOpts != nil {
		assetDbOpts.IncludeUserProgress = dbOpts.IncludeUserProgress
		assetDbOpts.IncludeAssetMetadata = dbOpts.IncludeAssetMetadata
	}

	assets, err := dao.ListAssets(ctx, assetDbOpts)
	if err != nil {
		return nil, err
	}

	assetMap := make(map[string][]*models.Asset)
	for _, a := range assets {
		assetMap[a.LessonID] = append(assetMap[a.LessonID], a)
	}

	// Stitch children onto parents in order
	for _, lesson := range lessons {
		lesson.Attachments = attMap[lesson.ID]
		lesson.Assets = assetMap[lesson.ID]
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
