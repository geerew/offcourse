package dao

// TODO Tidy to to make this more consistent. Use the builder pattern for all options
import (
	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/utils/pagination"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Options defines optional params for a database query
type Options struct {
	// ORDER BY
	//
	// Example: []string{"id DESC", "title ASC"}
	OrderBy []string

	// ORDER BY (clause)
	//
	// Example: []string{"id DESC", "title ASC"}
	OrderByClause squirrel.Sqlizer

	// Any valid squirrel WHERE expression
	//
	// Examples:
	//
	//   EQ:   squirrel.Eq{"id": "123"}
	//   IN:   squirrel.Eq{"id": []string{"123", "456"}}
	//   OR:   squirrel.Or{squirrel.Expr("id = ?", "123"), squirrel.Expr("id = ?", "456")}
	//   AND:  squirrel.And{squirrel.Eq{"id": "123"}, squirrel.Eq{"title": "devops"}}
	//   LIKE: squirrel.Like{"title": "%dev%"}
	//   NOT:  squirrel.NotEq{"id": "123"}
	Where squirrel.Sqlizer

	// IncludeUserProgress indicates whether to include course/asset progress
	// when performing a query
	IncludeUserProgress bool

	// IncludeAssetMetadata indicates whether to include asset metadata
	// when performing a query
	IncludeAssetMetadata bool

	// IncludeCourse indicates whether to include course table join
	IncludeCourse bool

	// IncludeLesson indicates whether to include lesson table join
	IncludeLesson bool

	// Used to paginate the results
	Pagination *pagination.Pagination
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewOptions creates an empty Options builder
func NewOptions() *Options {
	return &Options{}
}

// WithOrderBy appends ORDER BY fields
func (o *Options) WithOrderBy(fields ...string) *Options {
	o.OrderBy = append(o.OrderBy, fields...)
	return o
}

func (o *Options) OverrideOrderBy(fields ...string) *Options {
	o.OrderBy = fields
	return o
}

// WithOrderByClause sets a custom ORDER BY clause
//
// Use only if you need a complex ORDER BY that cannot be expressed with WithOrderBy
func (o *Options) WithOrderByClause(clause squirrel.Sqlizer) *Options {
	o.OrderByClause = clause
	return o
}

// WithWhere sets the WHERE clause using a squirrel.Sqlizer
func (o *Options) WithWhere(pred squirrel.Sqlizer) *Options {
	o.Where = pred
	return o
}

// WithCourse enables course table join
func (o *Options) WithCourse() *Options {
	o.IncludeCourse = true
	return o
}

// WithLesson enables lesson table join
func (o *Options) WithLesson() *Options {
	o.IncludeLesson = true
	return o
}

// WithPagination sets the pagination options
func (o *Options) WithPagination(p *pagination.Pagination) *Options {
	o.Pagination = p
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithUserProgress enables progress inclusion in queries
func (o *Options) WithUserProgress() *Options {
	o.IncludeUserProgress = true
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithAssetMetadata enables asset metadata inclusion in queries
func (o *Options) WithAssetMetadata() *Options {
	o.IncludeAssetMetadata = true
	return o
}
