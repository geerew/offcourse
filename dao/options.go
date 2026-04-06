package dao

import (
	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/utils/pagination"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Options defines a limited set of database query options
//
// This will passed to the internal DAO builder options struct for use in the query
// building process
type Options struct {
	// OrderBy can be used to order the results
	//
	// Example: []string{"id DESC", "title ASC"}
	OrderBy []string

	// OrderByClause can be used to set a custom ORDER BY clause, for example
	// when using a case expression
	OrderByClause squirrel.Sqlizer

	// Where is any valid squirrel WHERE expression
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

	// IncludeUserProgress indicates whether to include user progress when querying courses or
	// assets
	//
	// Valid when querying courses or assets
	IncludeUserProgress bool

	// IncludeAssetMetadata indicates whether to include asset metadata when querying assets
	//
	// Valid when querying assets
	IncludeAssetMetadata bool

	// Pagination is used to paginate the results
	//
	// Example: &pagination.Pagination{Page: 1, Limit: 10}
	Pagination *pagination.Pagination

	// StringQuery can be used to better filter the results. This will mainly only ever being used in the
	// API, which takes the query param from the client and parses it into a StringQuery struct
	//
	// Example:
	//   &StringQuery{
	//     Query: "tag:foo and progress:started or progress:completed",
	//     AllowedFilters: []string{"tag", "progress"},
	//   }
	StringQuery *StringQuery
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// StringQuery represents a list query, for example `q="tag:foo and progress:started or progress:completed"`
type StringQuery struct {
	Query          string
	AllowedFilters []string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewOptions creates an empty Options builder
func NewOptions() *Options {
	return &Options{}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithOrderBy appends ORDER BY fields
func (o *Options) WithOrderBy(fields ...string) *Options {
	o.OrderBy = append(o.OrderBy, fields...)
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (o *Options) OverrideOrderBy(fields ...string) *Options {
	o.OrderBy = fields
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithOrderByClause sets a custom ORDER BY clause
//
// Use only if you need a complex ORDER BY that cannot be expressed with WithOrderBy
func (o *Options) WithOrderByClause(clause squirrel.Sqlizer) *Options {
	o.OrderByClause = clause
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithWhere sets the WHERE clause using a squirrel.Sqlizer
func (o *Options) WithWhere(pred squirrel.Sqlizer) *Options {
	o.Where = pred
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithPagination sets the pagination options
func (o *Options) WithPagination(p *pagination.Pagination) *Options {
	o.Pagination = p
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithStringQuery sets a string query
func (o *Options) WithStringQuery(spec *StringQuery) *Options {
	o.StringQuery = spec
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithUserProgress enables progress inclusion in queries
//
// Can be used when querying assets and courses
//
//   - For assets, it adds an additional db query (asset progress)
//   - For courses, it adds 2 additional db queries (course progress and course favourite)
func (o *Options) WithUserProgress() *Options {
	o.IncludeUserProgress = true
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithAssetMetadata enables asset metadata inclusion in queries
//
// Can be used when querying assets and will add an additional db query to the asset metadata
// table
func (o *Options) WithAssetMetadata() *Options {
	o.IncludeAssetMetadata = true
	return o
}
