package queryparser

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryResult represents the result of a query parse
type QueryResult struct {
	Expr         QueryExpr
	Sort         []string
	FreeText     []string
	FoundFilters map[string]bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Parse a query string into an AST of sort tokens, free-text and allowed filters
// TODO support single quotes too
func Parse(q string, allowedFilters []string) (*QueryResult, error) {
	allTokens := tokenize(q)
	remainingTokens, sortTokens := extractSortTokens(allTokens)

	ast := newASTParser(remainingTokens, allowedFilters)
	expr, err := ast.parseOr()
	if err != nil {
		return nil, err
	}

	return &QueryResult{
		Expr:         expr,
		Sort:         sortTokens,
		FreeText:     ast.FreeText,
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
