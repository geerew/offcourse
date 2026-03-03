package dao

import (
	"fmt"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// countBuilder builds a query for counting distinct records in a table
func countBuilder(builderOpts builderOptions) (string, []interface{}, error) {
	if builderOpts.Table == "" {
		return "", nil, fmt.Errorf("builderOpts.Table cannot be empty")
	}

	builder := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Question)

	commonBuilder := builder.
		Select().
		From(builderOpts.Table)

	commonBuilder = applyJoins(commonBuilder, builderOpts.Joins)

	if builderOpts.DbOpts != nil {
		if builderOpts.DbOpts.Where != nil {
			commonBuilder = commonBuilder.Where(builderOpts.DbOpts.Where)
		}
	}

	// Fast path when no GROUP BY or HAVING
	if len(builderOpts.GroupBy) == 0 || builderOpts.Having == nil {
		return commonBuilder.
			Columns("COUNT(DISTINCT " + builderOpts.Table + ".id)").
			ToSql()
	}

	inner := commonBuilder.Columns(builderOpts.Table + ".id")

	if len(builderOpts.GroupBy) > 0 {
		inner = inner.GroupBy(builderOpts.GroupBy...)
	}

	if builderOpts.Having != nil {
		inner = inner.Having(builderOpts.Having)
	}

	return builder.
		Select("COUNT(*)").
		FromSelect(inner, "sub").
		ToSql()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// selectBuilder builds a query for selecting data from the database
func selectBuilder(builderOpts builderOptions) (string, []interface{}, error) {
	if builderOpts.Table == "" {
		return "", nil, fmt.Errorf("builderOpts.Table cannot be empty")
	}

	if len(builderOpts.Columns) == 0 {
		return "", nil, fmt.Errorf("builderOpts.Columns cannot be empty")
	}

	builder := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Question).
		Select(builderOpts.Columns...).
		From(builderOpts.Table)

	builder = applyJoins(builder, builderOpts.Joins)

	builder = builder.GroupBy(builderOpts.GroupBy...)

	if builderOpts.Having != nil {
		builder = builder.Having(builderOpts.Having)
	}

	// Database options
	if builderOpts.DbOpts != nil {
		builder = builder.Where(builderOpts.DbOpts.Where)

		builder = builder.OrderBy(builderOpts.DbOpts.OrderBy...)

		if builderOpts.DbOpts.OrderByClause != nil {
			builder = builder.OrderByClause(builderOpts.DbOpts.OrderByClause)
		}

		if builderOpts.DbOpts.Pagination != nil {
			builder = builder.
				Offset(uint64(builderOpts.DbOpts.Pagination.Offset())).
				Limit(uint64(builderOpts.DbOpts.Pagination.Limit()))
		}
	}

	return builder.ToSql()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// insertBuilder builds a query for inserting or replacing records in the database
func insertBuilder(builderOpts builderOptions) (string, []interface{}, error) {
	if builderOpts.Table == "" {
		return "", nil, fmt.Errorf("builderOpts.Table cannot be empty")
	}

	if len(builderOpts.Data) == 0 {
		return "", nil, fmt.Errorf("builderOpts.InsertData cannot be empty")
	}

	var builder squirrel.InsertBuilder
	if builderOpts.Replace {
		builder = squirrel.StatementBuilder.Replace(builderOpts.Table)
	} else {
		builder = squirrel.StatementBuilder.Insert(builderOpts.Table)
	}

	builder = builder.SetMap(builderOpts.Data)

	builder = builder.Suffix(builderOpts.Suffix)

	return builder.ToSql()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// updateBuilder builds a query for updating records in the database
//
// When BulkUpdate is set, it will fall through to the bulk update builder
func updateBuilder(builderOpts builderOptions) (string, []interface{}, error) {
	if builderOpts.Table == "" {
		return "", nil, fmt.Errorf("builderOpts.Table cannot be empty")
	}

	if builderOpts.BulkUpdate != nil && len(builderOpts.BulkUpdate.Rows) > 0 {
		return updateBuilderBulk(builderOpts)
	}

	if len(builderOpts.Data) == 0 {
		return "", nil, fmt.Errorf("builderOpts.Data cannot be empty")
	}

	if builderOpts.DbOpts == nil || builderOpts.DbOpts.Where == nil {
		return "", nil, fmt.Errorf("builderOpts.DbOpts.Where cannot be empty")
	}

	builder := squirrel.
		StatementBuilder.
		Update(builderOpts.Table).
		SetMap(builderOpts.Data).
		Where(builderOpts.DbOpts.Where)

	return builder.ToSql()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// updateBuilderBulk builds a query for updating multiple records in the database
func updateBuilderBulk(builderOpts builderOptions) (string, []interface{}, error) {
	bulkUpdate := builderOpts.BulkUpdate

	if bulkUpdate == nil {
		return "", nil, fmt.Errorf("builderOpts.BulkUpdate cannot be nil")
	}

	if len(bulkUpdate.Columns) == 0 {
		return "", nil, fmt.Errorf("builderOpts.BulkUpdate.Columns cannot be empty")
	}

	if len(bulkUpdate.Rows) == 0 {
		return "", nil, fmt.Errorf("builderOpts.BulkUpdate.Rows cannot be empty")
	}

	ids := utils.Map(bulkUpdate.Rows, func(row bulkUpdateRow) string { return row.ID })

	caseSQL := fmt.Sprintf("CASE %s %s END",
		models.BASE_ID,
		strings.TrimSpace(strings.Repeat("WHEN ? THEN ? ", len(bulkUpdate.Rows))),
	)

	setMap := make(map[string]interface{})
	for colIdx, col := range bulkUpdate.Columns {
		var caseArgs []interface{}
		for _, row := range bulkUpdate.Rows {
			if colIdx < len(row.Values) {
				caseArgs = append(caseArgs, row.ID, row.Values[colIdx])
			}
		}

		setMap[col] = squirrel.Expr(caseSQL, caseArgs...)
	}

	builder := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Question).
		Update(builderOpts.Table).
		SetMap(setMap).
		Where(squirrel.Eq{models.BASE_ID: ids})

	return builder.ToSql()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// deleteBuilder builds a query for deleting records from the database
func deleteBuilder(builderOpts builderOptions) (string, []interface{}, error) {
	if builderOpts.Table == "" {
		return "", nil, fmt.Errorf("builderOpts.Table cannot be empty")
	}

	builder := squirrel.
		StatementBuilder.
		Delete(builderOpts.Table)

	if builderOpts.DbOpts != nil && builderOpts.DbOpts.Where != nil {
		builder = builder.Where(builderOpts.DbOpts.Where)
	}

	return builder.ToSql()
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// applyJoins applies joins to a SELECT builder
func applyJoins(builder squirrel.SelectBuilder, joins []join) squirrel.SelectBuilder {
	for _, j := range joins {
		clause := fmt.Sprintf("%s ON %s", j.Table, j.Condition)
		switch j.Type {
		case joinTypeLeft:
			builder = builder.LeftJoin(clause)
		case joinTypeRight:
			builder = builder.RightJoin(clause)
		case joinTypeInner:
			builder = builder.Join(clause)
		}
	}
	return builder
}
