package queryparser

import (
	"fmt"
	"strings"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// astParser parses a slice of Tokens into an AST
type astParser struct {
	tokens         []token
	pos            int
	allowedFilters map[string]bool
	FoundFilters   map[string]bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// newASTParser creates a new astParser from a slice of tokens
func newASTParser(tokens []token, allowedFilters []string) *astParser {
	allowed := make(map[string]bool)
	found := make(map[string]bool)

	for _, f := range allowedFilters {
		allowed[f] = true
		found[f] = false
	}

	return &astParser{
		tokens:         tokens,
		pos:            0,
		allowedFilters: allowed,
		FoundFilters:   found,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// current returns the current token
func (ap *astParser) current() token {
	if ap.pos < len(ap.tokens) {
		return ap.tokens[ap.pos]
	}

	return token{Text: "", Quoted: false}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// consume returns the current token and advances the position
func (ap *astParser) consume() token {
	tok := ap.current()
	ap.pos++
	return tok
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// peek returns the next token without consuming it
func (ap *astParser) peek() token {
	if ap.pos < len(ap.tokens) {
		return ap.tokens[ap.pos]
	}

	return token{Text: "", Quoted: false}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (ap *astParser) makeFilter(key, val string) (*FilterExpr, error) {
	if !ap.allowedFilters[key] {
		return nil, fmt.Errorf("%w: unknown filter key %q", ErrInvalidSyntax, key)
	}
	val = strings.TrimSpace(val)
	if val == "" {
		if ap.pos < len(ap.tokens) && ap.current().Quoted {
			val = strings.TrimSpace(ap.consume().Text)
		}
	}
	if val == "" {
		return nil, fmt.Errorf("%w: empty value for filter %q", ErrInvalidSyntax, key)
	}
	ap.FoundFilters[key] = true
	return &FilterExpr{Key: key, Value: val}, nil
}

// parseOperand parses a single operand from the token slice.
// Every operand must be an allowed key:value filter (quoted value may follow key:).
func (ap *astParser) parseOperand() (QueryExpr, error) {
	if ap.pos >= len(ap.tokens) {
		return nil, nil
	}

	if ap.current().Text == "AND" || ap.current().Text == "OR" {
		ap.consume()
		return nil, nil
	}
	if ap.current().Text == "(" {
		ap.consume()

		expr, err := ap.parseOr()
		if err != nil {
			return nil, err
		}

		if ap.pos >= len(ap.tokens) || ap.current().Text != ")" {
			return nil, fmt.Errorf("expected ')'")
		}

		ap.consume()
		return expr, nil
	}

	cur := ap.current()

	if strings.Contains(cur.Text, ":") {
		parts := strings.SplitN(cur.Text, ":", 2)
		key := parts[0]
		val := strings.TrimSpace(parts[1])
		ap.consume()
		f, err := ap.makeFilter(key, val)
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	if cur.Quoted {
		return nil, fmt.Errorf("%w: quoted text must be a filter value after key:, use e.g. title:\"...\"", ErrInvalidSyntax)
	}

	var parts []string
	parts = append(parts, ap.consume().Text)
	for ap.pos < len(ap.tokens) {
		next := ap.peek()

		if next.Quoted || next.Text == "(" || next.Text == ")" || next.Text == "AND" || next.Text == "OR" {
			break
		}

		if strings.Contains(next.Text, ":") {
			candidate := strings.SplitN(next.Text, ":", 2)[0]
			if ap.allowedFilters[candidate] {
				break
			}
		}

		parts = append(parts, ap.consume().Text)
	}

	joined := strings.TrimSpace(strings.Join(parts, " "))
	if joined == "" {
		return nil, nil
	}

	if !strings.Contains(joined, ":") {
		return nil, fmt.Errorf("%w: expected key:value filter, got %q", ErrInvalidSyntax, joined)
	}

	parts2 := strings.SplitN(joined, ":", 2)
	key := parts2[0]
	rest := strings.TrimSpace(parts2[1])
	f, err := ap.makeFilter(key, rest)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseAnd parses a series of operands with an implicit AND between them
func (ap *astParser) parseAnd() (QueryExpr, error) {
	var children []QueryExpr

	for {
		operand, err := ap.parseOperand()
		if err != nil {
			return nil, err
		}
		if operand != nil {
			children = append(children, operand)
		}
		if ap.pos >= len(ap.tokens) {
			break
		}
		next := ap.peek()
		if strings.EqualFold(next.Text, "OR") || next.Text == ")" {
			break
		}
		if strings.EqualFold(next.Text, "AND") {
			ap.consume()
		}
	}

	if len(children) == 0 {
		return nil, nil
	}

	if len(children) == 1 {
		return children[0], nil
	}

	return &AndExpr{Children: children}, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseOr parses a series of And expressions separated by explicit OR
func (ap *astParser) parseOr() (QueryExpr, error) {
	expr, err := ap.parseAnd()
	if err != nil {
		return nil, err
	}

	for ap.pos < len(ap.tokens) {
		if !strings.EqualFold(ap.peek().Text, "OR") {
			break
		}

		ap.consume()

		right, err := ap.parseAnd()
		if err != nil {
			return nil, err
		}

		if right != nil {
			if expr == nil {
				expr = right
			} else {
				expr = &OrExpr{Children: []QueryExpr{expr, right}}
			}
		}
	}

	return expr, nil
}
