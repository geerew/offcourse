package queryparser

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryResult represents the result of a query parse
type QueryResult struct {
	Expr         QueryExpr
	FoundFilters map[string]bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Parse a query string into an AST of key:value filters combined with AND/OR and parentheses.
// Tokens must be allowed filter keys (see allowedFilters). There is no free-text mode.
func Parse(q string, allowedFilters []string) (*QueryResult, error) {
	allTokens, err := tokenize(q)
	if err != nil {
		return nil, err
	}

	ast := newASTParser(allTokens, allowedFilters)
	expr, err := ast.parseOr()
	if err != nil {
		return nil, err
	}

	return &QueryResult{
		Expr:         expr,
		FoundFilters: ast.FoundFilters,
	}, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsFilterWithKey checks if the given expression is a filter with the given key
func IsFilterWithKey(expr QueryExpr, key string) bool {
	if f, ok := expr.(*FilterExpr); ok {
		return f.Key == key
	}

	return false
}
