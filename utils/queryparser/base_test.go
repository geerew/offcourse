package queryparser

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var (
	allowedFilters = []string{"title", "available", "tag", "progress", "favourite"}
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestParse_Empty(t *testing.T) {
	q := ""
	result, err := Parse(q, allowedFilters)
	require.NoError(t, err)
	require.Nil(t, result.Expr)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestParse_FiltersOnly(t *testing.T) {
	t.Run("AND / OR / tags", func(t *testing.T) {
		q := `available:true AND tag:"go 1" OR progress:completed OR progress:"not started"`
		result, err := Parse(q, allowedFilters)
		require.NoError(t, err)

		require.Equal(t, "(((available:true AND tag:go 1) OR progress:completed) OR progress:not started)", result.Expr.String())
		require.True(t, result.FoundFilters["available"])
		require.True(t, result.FoundFilters["tag"])
		require.True(t, result.FoundFilters["progress"])
	})

	t.Run("title filter", func(t *testing.T) {
		q := `title:'go course' AND available:true`
		result, err := Parse(q, allowedFilters)
		require.NoError(t, err)
		require.Equal(t, "(title:go course AND available:true)", result.Expr.String())
		require.True(t, result.FoundFilters["title"])
		require.True(t, result.FoundFilters["available"])
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestParse_Rejections(t *testing.T) {
	t.Run("bare words", func(t *testing.T) {
		_, err := Parse("course 1 OR available:true", allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
	})

	t.Run("quoted without key", func(t *testing.T) {
		_, err := Parse(`"hello" AND available:true`, allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
	})

	t.Run("unknown filter key", func(t *testing.T) {
		_, err := Parse(`foo:bar`, allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
	})

	t.Run("unterminated quote", func(t *testing.T) {
		_, err := Parse(`title:"oops`, allowedFilters)
		require.Error(t, err)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestParse_SingleQuotedFilter(t *testing.T) {
	q := `tag:'tag 1' AND tag:"tag 2"`
	result, err := Parse(q, allowedFilters)
	require.NoError(t, err)
	require.Equal(t, "(tag:tag 1 AND tag:tag 2)", result.Expr.String())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestParse_Parentheses(t *testing.T) {
	q := "(tag:tag1 AND available:true) OR progress:completed"
	result, err := Parse(q, allowedFilters)
	require.NoError(t, err)
	require.Equal(t, "((tag:tag1 AND available:true) OR progress:completed)", result.Expr.String())
}
