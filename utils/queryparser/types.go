package queryparser

import "strings"

// QueryExpr is a filter literal or an AND/OR combination.
type QueryExpr interface {
	String() string
}

// FilterExpr is one predicate; Key is lowercased by the parser.
type FilterExpr struct {
	Key   string
	Value string
}

func (f *FilterExpr) String() string {
	return f.Key + ":" + f.Value
}

// AndExpr is implicit or explicit AND of two or more children.
type AndExpr struct {
	Children []QueryExpr
}

func (a *AndExpr) String() string {
	var parts []string
	for _, child := range a.Children {
		parts = append(parts, child.String())
	}
	return "(" + strings.Join(parts, " AND ") + ")"
}

// OrExpr is left-associative OR of exactly two children per node (nested for longer chains).
type OrExpr struct {
	Children []QueryExpr
}

func (o *OrExpr) String() string {
	var parts []string
	for _, child := range o.Children {
		parts = append(parts, child.String())
	}
	return "(" + strings.Join(parts, " OR ") + ")"
}
