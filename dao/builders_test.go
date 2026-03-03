package dao

import (
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_countBuilder(t *testing.T) {
	// Test successfully counting distinct records in a table
	t.Run("success", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE)
		sqlStr, args, err := countBuilder(*opts)
		require.NoError(t, err)
		require.Empty(t, args)
		require.Equal(t, "SELECT COUNT(DISTINCT assets.id) FROM assets", sqlStr)
	})

	// Test error due to empty table name
	t.Run("empty table", func(t *testing.T) {
		opts := newBuilderOptions("")
		_, _, err := countBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.Table cannot be empty", err.Error())
	})

	// Test successfully counting distinct records in a table with a where clause
	t.Run("with where", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			SetDbOpts(NewOptions().WithWhere(squirrel.Eq{models.ASSET_COURSE_ID: "course-1"}))
		sqlStr, args, err := countBuilder(*opts)
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, "course-1", args[0])
		require.Equal(t, "SELECT COUNT(DISTINCT assets.id) FROM assets WHERE course_id = ?", sqlStr)
	})

	// Test successfully counting distinct records in a table with a left join
	t.Run("with left join", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).WithLeftJoin("lessons l", "l.id = assets.lesson_id")
		sqlStr, _, err := countBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT COUNT(DISTINCT assets.id) FROM assets LEFT JOIN lessons l ON l.id = assets.lesson_id", sqlStr)
	})

	// Test successfully counting distinct records in a table with a right join
	t.Run("with right join", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).WithRightJoin("lessons l", "l.id = assets.lesson_id")
		sqlStr, _, err := countBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT COUNT(DISTINCT assets.id) FROM assets RIGHT JOIN lessons l ON l.id = assets.lesson_id", sqlStr)
	})

	// Test successfully counting distinct records in a table with an inner join
	t.Run("with inner join", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).WithJoin("lessons l", "l.id = assets.lesson_id")
		sqlStr, _, err := countBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT COUNT(DISTINCT assets.id) FROM assets JOIN lessons l ON l.id = assets.lesson_id", sqlStr)
	})

	// Test successfully counting distinct records in a table with a group by and having clause
	t.Run("with group by and having", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithGroupBy(models.ASSET_COURSE_ID).
			WithHaving(squirrel.Gt{"COUNT(*)": 1})
		sqlStr, args, err := countBuilder(*opts)
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, 1, args[0])
		require.Equal(t, "SELECT COUNT(*) FROM (SELECT assets.id FROM assets GROUP BY course_id HAVING COUNT(*) > ?) AS sub", sqlStr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_selectBuilder(t *testing.T) {
	// Test successfully selecting data from a table
	t.Run("success", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_TABLE_ID, models.ASSET_TABLE_TITLE)
		sqlStr, args, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Empty(t, args)
		require.Equal(t, "SELECT assets.id, assets.title FROM assets", sqlStr)
	})

	// Test error due to empty table name
	t.Run("empty table", func(t *testing.T) {
		opts := newBuilderOptions("").
			WithColumns("id")
		_, _, err := selectBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.Table cannot be empty", err.Error())
	})

	// Test error due to empty columns
	t.Run("empty columns", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE)
		_, _, err := selectBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.Columns cannot be empty", err.Error())
	})

	// Test successfully selecting data from a table with a where clause
	t.Run("with where", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_TABLE_ID).
			SetDbOpts(NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: "asset-1"}))
		sqlStr, args, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, "asset-1", args[0])
		require.Equal(t, "SELECT assets.id FROM assets WHERE assets.id = ?", sqlStr)
	})

	// Test successfully selecting data from a table with an order by clause
	t.Run("with orderby", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_TABLE_ID).
			SetDbOpts(NewOptions().WithOrderBy(models.ASSET_TABLE_CREATED_AT + " DESC"))
		sqlStr, _, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT assets.id FROM assets ORDER BY assets.created_at DESC", sqlStr)
	})

	// Test successfully selecting data from a table with a pagination clause
	t.Run("with pagination", func(t *testing.T) {
		p := pagination.New(1, 10)
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_TABLE_ID).
			SetDbOpts(NewOptions().WithPagination(p))
		sqlStr, _, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT assets.id FROM assets LIMIT 10 OFFSET 0", sqlStr)
	})

	// Test successfully selecting data from a table with a left join
	t.Run("with left join", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_TABLE_ID).
			WithLeftJoin("lessons l", "l.id = assets.lesson_id")
		sqlStr, _, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT assets.id FROM assets LEFT JOIN lessons l ON l.id = assets.lesson_id", sqlStr)
	})

	// Test successfully selecting data from a table with a right join
	t.Run("with right join", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_TABLE_ID).
			WithRightJoin("lessons l", "l.id = assets.lesson_id")
		sqlStr, _, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT assets.id FROM assets RIGHT JOIN lessons l ON l.id = assets.lesson_id", sqlStr)
	})

	// Test successfully selecting data from a table with an inner join
	t.Run("with inner join", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_TABLE_ID).
			WithJoin("lessons l", "l.id = assets.lesson_id")
		sqlStr, _, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT assets.id FROM assets JOIN lessons l ON l.id = assets.lesson_id", sqlStr)
	})

	// Test successfully selecting data from a table with group by and having
	t.Run("with group by and having", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_COURSE_ID, "COUNT(*) AS cnt").
			WithGroupBy(models.ASSET_COURSE_ID).
			WithHaving(squirrel.Gt{"COUNT(*)": 1})
		sqlStr, args, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, 1, args[0])
		require.Equal(t, "SELECT course_id, COUNT(*) AS cnt FROM assets GROUP BY course_id HAVING COUNT(*) > ?", sqlStr)
	})

	// Test successfully selecting data from a table with a custom order by clause
	t.Run("with order by clause", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithColumns(models.ASSET_TABLE_ID).
			SetDbOpts(NewOptions().WithOrderByClause(squirrel.Expr("assets.created_at DESC, assets.title ASC")))
		sqlStr, _, err := selectBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "SELECT assets.id FROM assets ORDER BY assets.created_at DESC, assets.title ASC", sqlStr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_insertBuilder(t *testing.T) {
	// Test successfully inserting data into a table
	t.Run("success", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithData(map[string]interface{}{
				models.BASE_ID:     "asset-1",
				models.ASSET_TITLE: "Asset 1",
			})
		sqlStr, args, err := insertBuilder(*opts)
		require.NoError(t, err)
		require.Len(t, args, 2)
		require.Equal(t, "INSERT INTO assets (id,title) VALUES (?,?) ", sqlStr)
		require.Equal(t, []interface{}{"asset-1", "Asset 1"}, args)
	})

	// Test error due to empty table name
	t.Run("empty table", func(t *testing.T) {
		opts := newBuilderOptions("").
			WithData(map[string]interface{}{"id": "x"})
		_, _, err := insertBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.Table cannot be empty", err.Error())
	})

	// Test error due to empty data
	t.Run("empty data", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE)
		_, _, err := insertBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.InsertData cannot be empty", err.Error())
	})

	// Test successfully inserting data into a table with a replace clause
	t.Run("replace", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithData(map[string]interface{}{models.BASE_ID: "asset-1"}).
			WithReplace()
		sqlStr, args, err := insertBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "REPLACE INTO assets (id) VALUES (?) ", sqlStr)
		require.Equal(t, []interface{}{"asset-1"}, args)
	})

	// Test successfully inserting data into a table with a suffix
	t.Run("with suffix", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithData(map[string]interface{}{models.BASE_ID: "asset-1"}).
			WithSuffix("ON CONFLICT(id) DO NOTHING")
		sqlStr, args, err := insertBuilder(*opts)
		require.NoError(t, err)
		require.Equal(t, "INSERT INTO assets (id) VALUES (?) ON CONFLICT(id) DO NOTHING", sqlStr)
		require.Equal(t, []interface{}{"asset-1"}, args)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_updateBuilder(t *testing.T) {
	// Test successfully updating data in a table
	t.Run("success", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithData(map[string]interface{}{models.ASSET_TITLE: "Updated"}).
			SetDbOpts(NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: "asset-1"}))
		sqlStr, args, err := updateBuilder(*opts)
		require.NoError(t, err)
		require.Len(t, args, 2)
		require.Equal(t, "UPDATE assets SET title = ? WHERE id = ?", sqlStr)
		require.Equal(t, []interface{}{"Updated", "asset-1"}, args)
	})

	// Test error due to empty table name
	t.Run("empty table", func(t *testing.T) {
		opts := newBuilderOptions("").
			WithData(map[string]interface{}{"title": "x"}).
			SetDbOpts(NewOptions().WithWhere(squirrel.Eq{"id": "1"}))
		_, _, err := updateBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.Table cannot be empty", err.Error())
	})

	// Test error due to empty data
	t.Run("empty data", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			SetDbOpts(NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: "asset-1"}))
		_, _, err := updateBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.Data cannot be empty", err.Error())
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			WithData(map[string]interface{}{models.ASSET_TITLE: "Updated"})
		_, _, err := updateBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.DbOpts.Where cannot be empty", err.Error())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_deleteBuilder(t *testing.T) {
	// Test successfully deleting data from a table
	t.Run("success", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE).
			SetDbOpts(NewOptions().WithWhere(squirrel.Eq{models.BASE_ID: "asset-1"}))
		sqlStr, args, err := deleteBuilder(*opts)
		require.NoError(t, err)
		require.Len(t, args, 1)
		require.Equal(t, "DELETE FROM assets WHERE id = ?", sqlStr)
		require.Equal(t, []interface{}{"asset-1"}, args)
	})

	// Test error due to empty table name
	t.Run("empty table", func(t *testing.T) {
		opts := newBuilderOptions("").
			SetDbOpts(NewOptions().WithWhere(squirrel.Eq{"id": "1"}))
		_, _, err := deleteBuilder(*opts)
		require.Error(t, err)
		require.Equal(t, "builderOpts.Table cannot be empty", err.Error())
	})

	// Test successfully deleting data from a table without a where clause
	t.Run("without where", func(t *testing.T) {
		opts := newBuilderOptions(models.ASSET_TABLE)
		sqlStr, args, err := deleteBuilder(*opts)
		require.NoError(t, err)
		require.Empty(t, args)
		require.Equal(t, "DELETE FROM assets", sqlStr)
	})
}
