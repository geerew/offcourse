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

func TestParse_Success(t *testing.T) {
	// Test successfully parsing a query with AND / OR / tags
	t.Run("AND / OR / tags", func(t *testing.T) {
		q := `available:true AND tag:"go 1" OR progress:completed OR progress:"not started"`
		result, err := Parse(q, allowedFilters)
		require.NoError(t, err)

		require.Equal(t, "(((available:true AND tag:go 1) OR progress:completed) OR progress:not started)", result.Expr.String())
		require.True(t, result.FoundFilter("available"))
		require.True(t, result.FoundFilter("tag"))
		require.True(t, result.FoundFilter("progress"))
	})

	// Test successfully parsing a query with parentheses
	t.Run("parentheses", func(t *testing.T) {
		q := "(tag:tag1 AND available:true) OR progress:completed"
		result, err := Parse(q, allowedFilters)
		require.NoError(t, err)

		require.Equal(t, "((tag:tag1 AND available:true) OR progress:completed)", result.Expr.String())
		require.True(t, result.FoundFilter("tag"))
		require.True(t, result.FoundFilter("available"))
		require.True(t, result.FoundFilter("progress"))

	})

	// Test successfully parsing a query with quoted tags
	t.Run("quoted tags", func(t *testing.T) {
		q := `tag:'tag 1' AND tag:"tag 2"`
		result, err := Parse(q, allowedFilters)
		require.NoError(t, err)

		require.Equal(t, "(tag:tag 1 AND tag:tag 2)", result.Expr.String())
		require.True(t, result.FoundFilter("tag"))
	})

	// Test successfully parsing a query with normalized keys
	t.Run("normalized keys", func(t *testing.T) {
		q := "Available:true AND Progress:completed"
		result, err := Parse(q, allowedFilters)
		require.NoError(t, err)

		require.Equal(t, "(available:true AND progress:completed)", result.Expr.String())
		require.True(t, result.FoundFilter("available"))
		require.True(t, result.FoundFilter("progress"))
	})

	// Test successfully parsing a query with escaped quotes in value
	t.Run("escaped quotes in value", func(t *testing.T) {
		q := `tag:"say \"hi\"" AND available:true`
		result, err := Parse(q, allowedFilters)
		require.NoError(t, err)
		require.Equal(t, "(tag:say \"hi\" AND available:true)", result.Expr.String())
	})

	// Test successfully parsing a query with lowercase and/or from lexer
	t.Run("lowercase and/or from lexer", func(t *testing.T) {
		q := "available:true or tag:x"
		result, err := Parse(q, allowedFilters)
		require.NoError(t, err)
		require.Equal(t, "(available:true OR tag:x)", result.Expr.String())
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestParse_Errors(t *testing.T) {
	// Test error due to bare words
	t.Run("bare words", func(t *testing.T) {
		q := "course 1 OR available:true"
		_, err := Parse(q, allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
		require.ErrorIs(t, err, ErrExpectedKeyValue)
	})

	t.Run("trailing tokens", func(t *testing.T) {
		q := "available:true )"
		_, err := Parse(q, allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
		require.ErrorIs(t, err, ErrTrailingInput)
	})

	// Test error due to quoted without key
	t.Run("quoted without key", func(t *testing.T) {
		q := `"hello" AND available:true`
		_, err := Parse(q, allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
		require.ErrorIs(t, err, ErrUnexpectedQuotedToken)
	})

	// Test error due to unknown filter key
	t.Run("unknown filter key", func(t *testing.T) {
		q := `foo:bar`
		_, err := Parse(q, allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
		require.ErrorIs(t, err, ErrUnknownFilterKey)
	})

	// Test error due to unterminated quote
	t.Run("unterminated quote", func(t *testing.T) {
		q := `title:"oops`
		_, err := Parse(q, allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
		require.ErrorIs(t, err, ErrUnterminatedQuote)
	})

	// Test error due to sand is not AND keyword
	t.Run("sand is not AND keyword", func(t *testing.T) {
		_, err := Parse("sand:x AND available:true", allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrUnknownFilterKey)
	})

	// Test error due to android is not AND keyword
	t.Run("android is not AND keyword", func(t *testing.T) {
		_, err := Parse("progress:completed android available:true", allowedFilters)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrInvalidSyntax)
		require.ErrorIs(t, err, ErrExpectedKeyValue)
	})
}
