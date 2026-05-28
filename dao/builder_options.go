package dao

// TODO Tidy to to make this more consistent. Use the builder pattern for all options
import (
	"github.com/Masterminds/squirrel"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// joinType represents a join type
type joinType string

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// joinType constants
const (
	joinTypeInner joinType = "INNER JOIN"
	joinTypeLeft  joinType = "LEFT JOIN"
	joinTypeRight joinType = "RIGHT JOIN"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// join defines a join in a SQL query (internal to DAO)
type join struct {
	Type      joinType
	Table     string
	Condition string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// bulkUpdate defines a bulk update
type bulkUpdate struct {
	Columns []string
	Rows    []bulkUpdateRow
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// bulkUpdateRow defines a row in a bulk update
type bulkUpdateRow struct {
	ID     string
	Values []interface{}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// builderOptions defines builder options for a database query
type builderOptions struct {
	// The name of the table to query
	//
	// Example: "table1"
	Table string

	// Columns to select
	//
	// Example: []string{"id", "title", "created_at"}
	Columns []string

	// Data is a key/value map of data to insert into the table during an INSERT or
	// UPDATE
	//
	// Example: map[string]interface{}{"id": "123", "title": "Test", "created_at": time.Now()}
	Data map[string]interface{}

	// Columns to group by
	//
	// Example: []string{table1.id}
	GroupBy []string

	// Having clause (used in conjunction with GroupBy)
	//
	// Example: squirrel.Eq{"COUNT(table1.id)": 1}
	Having squirrel.Sqlizer

	// Joins to use in SELECT queries
	//
	// Example: []join{{Type: joinTypeInner, Table: "table1", Condition: "table1.id = table2.id"}}
	Joins []join

	// Suffix is raw SQL to append to the query
	//
	// Example: "ON CONFLICT(id) DO NOTHING"
	Suffix string

	// Limit for the number of results to return
	Limit int

	// Whether to use REPLACE INTO instead of INSERT INTO
	Replace bool

	// BulkUpdate configures a bulk UPDATE
	BulkUpdate *bulkUpdate

	// Database options
	//
	// Example: &Options{Where: squirrel.Eq{"id": "123"}}
	DbOpts *Options
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// newBuilderOptions creates an new builderOptions instance with the table name set
func newBuilderOptions(table string) *builderOptions {
	return &builderOptions{Table: table}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithColumns sets the columns to select
//
// Can be called multiple times to add additional columns
func (o *builderOptions) WithColumns(columns ...string) *builderOptions {
	o.Columns = append(o.Columns, columns...)
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithData sets the data to insert
//
// Can be called multiple times to add additional data. The data will be merged
func (o *builderOptions) WithData(data map[string]interface{}) *builderOptions {
	if o.Data == nil {
		o.Data = make(map[string]interface{})
	}

	for key, value := range data {
		o.Data[key] = value
	}

	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithGroupBy appends GROUP BY fields
//
// Can be called multiple times to add additional GROUP BY fields
func (o *builderOptions) WithGroupBy(fields ...string) *builderOptions {
	o.GroupBy = append(o.GroupBy, fields...)
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithHaving sets the HAVING clause using a squirrel.Sqlizer
//
// # WithHaving should be used in conjunction with WithGroupBy
//
// Use once per builderOptions instance
func (o *builderOptions) WithHaving(pred squirrel.Sqlizer) *builderOptions {
	o.Having = pred
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithJoin appends an INNER JOIN clause
//
// Can be called multiple times to add multiple joins
func (o *builderOptions) WithJoin(table, condition string) *builderOptions {
	o.Joins = append(o.Joins, join{Type: joinTypeInner, Table: table, Condition: condition})
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithLeftJoin appends a LEFT JOIN clause
//
// Can be called multiple times to add multiple joins
func (o *builderOptions) WithLeftJoin(table, onCondition string) *builderOptions {
	o.Joins = append(o.Joins, join{Type: joinTypeLeft, Table: table, Condition: onCondition})
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithRightJoin appends a RIGHT JOIN clause
//
// Can be called multiple times to add multiple joins
func (o *builderOptions) WithRightJoin(table, condition string) *builderOptions {
	o.Joins = append(o.Joins, join{Type: joinTypeRight, Table: table, Condition: condition})
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithSuffix registers raw SQL to append via squirrel.Suffix(...)
//
// Use once per builderOptions instance
func (o *builderOptions) WithSuffix(sql string) *builderOptions {
	o.Suffix = sql
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithLimit sets the limit for the number of results to return
//
// Use once per builderOptions instance
func (o *builderOptions) WithLimit(limit int) *builderOptions {
	o.Limit = limit
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithReplace sets the builder to use REPLACE INTO instead of INSERT INTO
func (o *builderOptions) WithReplace() *builderOptions {
	o.Replace = true
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SetDbOpts sets the database options
//
// Use once per builderOptions instance
func (o *builderOptions) SetDbOpts(opts *Options) *builderOptions {
	if opts == nil {
		o.DbOpts = NewOptions()
	} else {
		o.DbOpts = opts
	}

	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithBulkUpdate configures a bulk UPDATE
//
// Use in conjunction with WithBulkUpdateRow
func (o *builderOptions) WithBulkUpdate(columns ...string) *builderOptions {
	o.BulkUpdate = &bulkUpdate{
		Columns: columns,
		Rows:    nil,
	}
	return o
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// WithBulkUpdateRow appends a row to the bulk update
//
// Use in conjunction with WithBulkUpdate
func (o *builderOptions) WithBulkUpdateRow(id string, values ...interface{}) *builderOptions {
	if o.BulkUpdate == nil {
		return o
	}
	o.BulkUpdate.Rows = append(o.BulkUpdate.Rows, bulkUpdateRow{ID: id, Values: values})
	return o
}
