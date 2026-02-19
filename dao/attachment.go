package dao

import (
	"context"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateAttachment inserts a new attachment record
func (dao *DAO) CreateAttachment(ctx context.Context, attachment *models.Attachment) error {
	if attachment == nil {
		return utils.ErrNilPtr
	}

	if attachment.ID == "" {
		attachment.RefreshId()
	}

	if attachment.Title == "" {
		return utils.ErrTitle
	}

	if attachment.Path == "" {
		return utils.ErrPath
	}

	attachment.RefreshCreatedAt()
	attachment.RefreshUpdatedAt()

	builderOpts := newBuilderOptions(models.ATTACHMENT_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:              attachment.ID,
				models.ATTACHMENT_LESSON_ID: attachment.LessonID,
				models.ATTACHMENT_TITLE:     attachment.Title,
				models.ATTACHMENT_PATH:      attachment.Path,
				models.BASE_CREATED_AT:      attachment.CreatedAt,
				models.BASE_UPDATED_AT:      attachment.UpdatedAt,
			},
		)

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetAttachment gets a record from the attachments table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
func (dao *DAO) GetAttachment(ctx context.Context, dbOpts *Options) (*models.Attachment, error) {
	builderOpts := newBuilderOptions(models.ATTACHMENT_TABLE).
		WithColumns(models.AttachmentColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	// Add lesson and course joins if enabled
	if dbOpts != nil {
		if dbOpts.IncludeLesson {
			builderOpts = builderOpts.
				WithJoin(models.LESSON_TABLE, models.ATTACHMENT_TABLE_LESSON_ID+" = "+models.LESSON_TABLE_ID)

			// If course is also enabled, join course through lesson
			if dbOpts.IncludeCourse {
				builderOpts = builderOpts.
					WithJoin(models.COURSE_TABLE, models.LESSON_TABLE_COURSE_ID+" = "+models.COURSE_TABLE_ID)
			}
		}
	}

	return getGeneric[models.Attachment](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListAttachments gets all records from the attachments table based upon the where clause and pagination
// in the options
func (dao *DAO) ListAttachments(ctx context.Context, dbOpts *Options) ([]*models.Attachment, error) {
	builderOpts := newBuilderOptions(models.ATTACHMENT_TABLE).
		WithColumns(models.AttachmentColumns()...).
		SetDbOpts(dbOpts)

	// Add lesson and course joins if enabled
	if dbOpts != nil {
		if dbOpts.IncludeLesson {
			builderOpts = builderOpts.
				WithJoin(models.LESSON_TABLE, models.ATTACHMENT_TABLE_LESSON_ID+" = "+models.LESSON_TABLE_ID)

			// If course is also enabled, join course through lesson
			if dbOpts.IncludeCourse {
				builderOpts = builderOpts.
					WithJoin(models.COURSE_TABLE, models.LESSON_TABLE_COURSE_ID+" = "+models.COURSE_TABLE_ID)
			}
		}
	}

	return listGeneric[models.Attachment](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UpdateAttachment updates an attachment record
func (dao *DAO) UpdateAttachment(ctx context.Context, attachment *models.Attachment) error {
	if attachment == nil {
		return utils.ErrNilPtr
	}

	if attachment.ID == "" {
		return utils.ErrId
	}

	if attachment.Title == "" {
		return utils.ErrTitle
	}

	if attachment.Path == "" {
		return utils.ErrPath
	}

	attachment.RefreshUpdatedAt()

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: attachment.ID})

	builderOpts := newBuilderOptions(models.ATTACHMENT_TABLE).
		WithData(
			map[string]interface{}{
				models.ATTACHMENT_TITLE: attachment.Title,
				models.ATTACHMENT_PATH:  attachment.Path,
				models.BASE_UPDATED_AT:  attachment.UpdatedAt,
			},
		).
		SetDbOpts(dbOpts)

	_, err := updateGeneric(ctx, dao, *builderOpts)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteAttachments deletes records from the attachments table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteAttachments(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.ATTACHMENT_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}
