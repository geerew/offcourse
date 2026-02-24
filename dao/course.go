package dao

import (
	"context"
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
// Course progress is not included by default. It can be enabled by calling `WithUserProgress()` on the options
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

	// Validate principal early
	principal, err := principalFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	course, err := getGeneric[models.Course](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if course == nil {
		return nil, nil
	}

	// Load progress and favourite separately
	if err := attachCourseProgressAndFavourite(ctx, dao, principal.UserID, []*models.Course{course}); err != nil {
		return nil, err
	}

	return course, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// ListCourses gets all records from the courses table based upon the where clause and pagination
// in the options
//
// Course progress and favourite status are not included by default. They can be enabled by calling `WithUserProgress()` on the options
func (dao *DAO) ListCourses(ctx context.Context, dbOpts *Options) ([]*models.Course, error) {
	builderOpts := newBuilderOptions(models.COURSE_TABLE).
		WithColumns(models.CourseColumns()...).
		SetDbOpts(dbOpts)

	includeProgress := dbOpts != nil && dbOpts.IncludeUserProgress

	// When progress is not included, use a simpler query
	if !includeProgress {
		return listGeneric[models.Course](ctx, dao, *builderOpts)
	}

	// Validate principal early
	principal, err := principalFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	courses, err := listGeneric[models.Course](ctx, dao, *builderOpts)
	if err != nil {
		return nil, err
	}

	if len(courses) == 0 {
		return nil, nil
	}

	// Load progress and favourite separately
	if err := attachCourseProgressAndFavourite(ctx, dao, principal.UserID, courses); err != nil {
		return nil, err
	}

	return courses, nil
}

// attachCourseProgressAndFavourite fetches progress and favourites for the given courses and attaches
// them. This centralizes the logic so adding new relations (e.g. favourites) only requires updating
// this helper.
func attachCourseProgressAndFavourite(ctx context.Context, dao *DAO, userID string, courses []*models.Course) error {
	if len(courses) == 0 {
		return nil
	}

	courseIDs := make([]string, len(courses))
	for i, c := range courses {
		courseIDs[i] = c.ID
	}

	// Batch fetch progress
	progressOpts := NewOptions().WithWhere(squirrel.And{
		squirrel.Eq{models.COURSE_PROGRESS_USER_ID: userID},
		squirrel.Eq{models.COURSE_PROGRESS_COURSE_ID: courseIDs},
	})
	progressList, err := dao.ListCourseProgress(ctx, progressOpts)
	if err != nil {
		return err
	}
	progressMap := make(map[string]*models.CourseProgress)
	for _, p := range progressList {
		progressMap[p.CourseID] = p
	}

	// Batch fetch favourites
	favouriteOpts := NewOptions().WithWhere(squirrel.And{
		squirrel.Eq{models.COURSE_FAVOURITE_USER_ID: userID},
		squirrel.Eq{models.COURSE_FAVOURITE_COURSE_ID: courseIDs},
	})
	favouritesList, err := dao.ListCourseFavourites(ctx, favouriteOpts)
	if err != nil {
		return err
	}
	favouritedSet := make(map[string]bool)
	for _, f := range favouritesList {
		favouritedSet[f.CourseID] = true
	}

	// Attach to each course
	for _, c := range courses {
		if p, ok := progressMap[c.ID]; ok {
			c.Progress = p
		} else {
			c.Progress = &models.CourseProgress{CourseID: c.ID, UserID: userID}
		}
		c.Favourited = favouritedSet[c.ID]
	}

	return nil
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
