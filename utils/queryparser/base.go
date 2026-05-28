package queryparser

import "strings"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// QueryResult holds the parsed expression and which allowed keys occurred at least once.
type QueryResult struct {
	Expr         QueryExpr
	foundFilters map[string]bool
}

// FoundFilter reports whether key occurred in the query (lookup is lowercased + trimmed)
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

// Parse turns q into an expression tree: only allowed key:value filters, AND/OR, parentheses.
// Keys are lowercased; empty q yields nil Expr. Parsing fails if any input remains unconsumed.
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

	foundCopy := make(map[string]bool, len(p.foundFilters))
	for k, v := range p.foundFilters {
		foundCopy[k] = v
	}

	return &QueryResult{
		Expr:         expr,
		foundFilters: foundCopy,
	}, nil
}
