package queryparser

import "strings"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryResult represents the result of a query parse
type QueryResult struct {
	Expr         QueryExpr
	foundFilters map[string]bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// FoundFilter returns true when a filter was found at least once
func (r *QueryResult) FoundFilter(key string) bool {
	if r == nil || r.foundFilters == nil {
		return false
	}

	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return false
	}

	return r.foundFilters[k]
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Parse a query string into an AST of key:value filters combined with AND/OR and parentheses

func Parse(q string, allowedKeys []string) (*QueryResult, error) {
	allTokens, err := tokenize(q)
	if err != nil {
		return nil, err
	}

	p := newParser(allTokens, allowedKeys)
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}

	if err := p.assertFullyConsumed(); err != nil {
		return nil, err
	}

	return &QueryResult{
		Expr:         expr,
		foundFilters: p.FoundFilters,
	}, nil
}
