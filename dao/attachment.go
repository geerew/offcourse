package dao

import (
	"context"

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

	if attachment.CourseID == "" {
		return utils.ErrCourseId
	}

	if attachment.LessonID == "" {
		return utils.ErrLessonId
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
				models.ATTACHMENT_COURSE_ID: attachment.CourseID,
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

	return getGeneric[models.Attachment](ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListAttachments gets all records from the attachments table based upon the where clause and pagination
// in the options
func (dao *DAO) ListAttachments(ctx context.Context, dbOpts *Options) ([]*models.Attachment, error) {
	if err := applyStringQuerySortOnly(dbOpts); err != nil {
		return nil, err
	}

	builderOpts := newBuilderOptions(models.ATTACHMENT_TABLE).
		WithColumns(models.AttachmentColumns()...).
		SetDbOpts(dbOpts)

	return listGeneric[models.Attachment](ctx, dao, *builderOpts)
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
