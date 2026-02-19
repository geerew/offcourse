package dao

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// CreateCourse inserts a new course record
func (dao *DAO) CreateCourse(ctx context.Context, course *models.Course) error {
	if course == nil {
		return utils.ErrNilPtr
	}

	if course.Title == "" {
		return utils.ErrTitle
	}

	if course.Path == "" {
		return utils.ErrPath
	}

	if course.ID == "" {
		course.RefreshId()
	}

	course.RefreshCreatedAt()
	course.RefreshUpdatedAt()

	// Ensure initial scan is false and maintenance is true
	course.InitialScan = false
	course.Maintenance = true

	builderOpts := newBuilderOptions(models.COURSE_TABLE).
		WithData(
			map[string]interface{}{
				models.BASE_ID:              course.ID,
				models.COURSE_TITLE:         course.Title,
				models.COURSE_PATH:          course.Path,
				models.COURSE_CARD_PATH:     course.CardPath,
				models.COURSE_CARD_HASH:     course.CardHash,
				models.COURSE_CARD_MOD_TIME: course.CardModTime,
				models.COURSE_AVAILABLE:     course.Available,
				models.COURSE_DURATION:      course.Duration,
				models.COURSE_INITIAL_SCAN:  course.InitialScan,
				models.COURSE_MAINTENANCE:   course.Maintenance,
				models.COURSE_DESCRIPTION:   course.Description,
				models.BASE_CREATED_AT:      course.CreatedAt,
				models.BASE_UPDATED_AT:      course.UpdatedAt,
			},
		)

	return createGeneric(ctx, dao, *builderOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetCourse gets a record from the courses table based upon the where clause in the options. If
// there is no where clause, it will return the first record in the table
//
// By default, progress is not included. Use `WithUserProgress()` on the options to include it
func (dao *DAO) GetCourse(ctx context.Context, dbOpts *Options) (*models.Course, error) {
	builderOpts := newBuilderOptions(models.COURSE_TABLE).
		WithColumns(models.CourseColumns()...).
		SetDbOpts(dbOpts).
		WithLimit(1)

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress

	// When progress is not included, use a simpler query
	if !includeProgress {
		return getGeneric[models.Course](ctx, dao, *builderOpts)
	}

	// Include progress in the query
	principal, err := principalFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	builderOpts = builderOpts.
		WithColumns(models.CourseProgressRowColumns()...).
		WithColumns(fmt.Sprintf("%s AS favourite_id", models.COURSE_FAVOURITE_TABLE_ID)).
		WithLeftJoin(models.COURSE_PROGRESS_TABLE, fmt.Sprintf("%s = %s AND %s = '%s'", models.COURSE_PROGRESS_TABLE_COURSE_ID, models.COURSE_TABLE_ID, models.COURSE_PROGRESS_TABLE_USER_ID, principal.UserID)).
		WithLeftJoin(models.COURSE_FAVOURITE_TABLE, fmt.Sprintf("%s = %s AND %s = '%s'", models.COURSE_FAVOURITE_TABLE_COURSE_ID, models.COURSE_TABLE_ID, models.COURSE_FAVOURITE_TABLE_USER_ID, principal.UserID))

	row, err := getGeneric[models.CourseRow](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if row == nil {
		return nil, nil
	}

	return row.ToDomain(), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListCourses gets all records from the courses table based upon the where clause and pagination
// in the options
//
// By default, progress is not included. Use `WithUserProgress()` on the options to include it
func (dao *DAO) ListCourses(ctx context.Context, dbOpts *Options) ([]*models.Course, error) {
	builderOpts := newBuilderOptions(models.COURSE_TABLE).
		WithColumns(models.CourseColumns()...).
		SetDbOpts(dbOpts)

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress

	// When progress is not included, use a simpler query
	if !includeProgress {
		return listGeneric[models.Course](ctx, dao, *builderOpts)
	}

	// Include progress in the query
	principal, err := principalFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	builderOpts = builderOpts.
		WithColumns(models.CourseProgressRowColumns()...).
		WithColumns(fmt.Sprintf("%s AS favourite_id", models.COURSE_FAVOURITE_TABLE_ID)).
		WithLeftJoin(models.COURSE_PROGRESS_TABLE, fmt.Sprintf("%s = %s AND %s = '%s'", models.COURSE_PROGRESS_TABLE_COURSE_ID, models.COURSE_TABLE_ID, models.COURSE_PROGRESS_TABLE_USER_ID, principal.UserID)).
		WithLeftJoin(models.COURSE_FAVOURITE_TABLE, fmt.Sprintf("%s = %s AND %s = '%s'", models.COURSE_FAVOURITE_TABLE_COURSE_ID, models.COURSE_TABLE_ID, models.COURSE_FAVOURITE_TABLE_USER_ID, principal.UserID))

	rows, err := listGeneric[models.CourseRow](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	records := make([]*models.Course, 0, len(rows))
	for i := range rows {
		r := rows[i]
		records = append(records, r.ToDomain())
	}

	return records, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// UpdateCourse updates a course record
func (dao *DAO) UpdateCourse(ctx context.Context, course *models.Course) error {
	if course == nil {
		return utils.ErrNilPtr
	}

	if course.ID == "" {
		return utils.ErrId
	}

	if course.Title == "" {
		return utils.ErrTitle
	}

	if course.Path == "" {
		return utils.ErrPath
	}

	course.RefreshUpdatedAt()

	dbOpts := NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: course.ID})

	builderOpts := newBuilderOptions(models.COURSE_TABLE).
		WithData(
			map[string]interface{}{
				models.COURSE_TITLE:         course.Title,
				models.COURSE_PATH:          course.Path,
				models.COURSE_CARD_PATH:     course.CardPath,
				models.COURSE_CARD_HASH:     course.CardHash,
				models.COURSE_CARD_MOD_TIME: course.CardModTime,
				models.COURSE_AVAILABLE:     course.Available,
				models.COURSE_DURATION:      course.Duration,
				models.COURSE_INITIAL_SCAN:  course.InitialScan,
				models.COURSE_MAINTENANCE:   course.Maintenance,
				models.COURSE_DESCRIPTION:   course.Description,
				models.BASE_UPDATED_AT:      course.UpdatedAt,
			},
		).
		SetDbOpts(dbOpts)

	_, err := updateGeneric(ctx, dao, *builderOpts)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteCourses deletes records from the courses table
//
// Errors when a where clause is not provided
func (dao *DAO) DeleteCourses(ctx context.Context, dbOpts *Options) error {
	if dbOpts == nil || dbOpts.Where == nil {
		return utils.ErrWhere
	}

	builderOpts := newBuilderOptions(models.COURSE_TABLE).SetDbOpts(dbOpts)
	sqlStr, args, _ := deleteBuilder(*builderOpts)

	_, err := dao.db.ExecContext(ctx, sqlStr, args...)
	return err
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ClassifyCoursePaths classifies the given paths into one of the following categories:
//   - PathClassificationNone: The path does not exist in the courses table
//   - PathClassificationAncestor: The path is an ancestor of a course path
//   - PathClassificationCourse: The path is an exact match to a course path
//   - PathClassificationDescendant: The path is a descendant of a course path
//
// The paths are returned as a map with the original path as the key and the classification as the
// value
func (dao *DAO) ClassifyCoursePaths(ctx context.Context, paths []string) (map[string]types.PathClassification, error) {
	paths = slices.DeleteFunc(paths, func(s string) bool {
		return s == ""
	})

	if len(paths) == 0 {
		return nil, nil
	}

	results := make(map[string]types.PathClassification)
	for _, path := range paths {
		results[path] = types.PathClassificationNone
	}

	whereClause := make([]squirrel.Sqlizer, len(paths))
	for i, path := range paths {
		whereClause[i] = squirrel.Like{models.COURSE_TABLE_PATH: path + "%"}
	}

	query, args, _ := squirrel.
		StatementBuilder.
		Select(models.COURSE_TABLE_PATH).
		From(models.COURSE_TABLE).
		Where(squirrel.Or(whereClause)).
		ToSql()

	rows, err := dao.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coursePath string
	coursePaths := []string{}
	for rows.Next() {
		if err := rows.Scan(&coursePath); err != nil {
			return nil, err
		}
		coursePaths = append(coursePaths, coursePath)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Process
	for _, path := range paths {
		for _, coursePath := range coursePaths {
			if coursePath == path {
				results[path] = types.PathClassificationCourse
				break
			} else if strings.HasPrefix(coursePath, path) {
				results[path] = types.PathClassificationAncestor
				break
			} else if strings.HasPrefix(path, coursePath) && results[path] != types.PathClassificationAncestor {
				results[path] = types.PathClassificationDescendant
				break
			}
		}
	}

	return results, nil
}
