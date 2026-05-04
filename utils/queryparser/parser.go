package queryparser

import (
	"errors"
	"fmt"
	"strings"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parser parses a slice of Tokens into an AST
type parser struct {
	tokens       []token
	pos          int
	allowedKeys  map[string]struct{}
	FoundFilters map[string]bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// newParser creates a parser
func newParser(tokens []token, allowedKeys []string) *parser {
	allowed := make(map[string]struct{})
	found := make(map[string]bool)

	for _, key := range allowedKeys {
		if key == "" {
			continue
		}

		normalizedKey := strings.ToLower(strings.TrimSpace(key))
		allowed[normalizedKey] = struct{}{}
		found[normalizedKey] = false
	}

	return &parser{
		tokens:       tokens,
		pos:          0,
		allowedKeys:  allowed,
		FoundFilters: found,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// current returns the current token
func (p *parser) current() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}

	return token{Text: "", Quoted: false}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// consume returns the current token and advances the position
func (p *parser) consume() token {
	tok := p.current()
	p.pos++
	return tok
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// peek returns the next token without consuming it
func (p *parser) peek() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}

	return token{Text: "", Quoted: false}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (p *parser) makeFilter(key, val string) (*FilterExpr, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if _, ok := p.allowedKeys[key]; !ok {
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: %q", ErrUnknownFilterKey, key))
	}
	val = strings.TrimSpace(val)
	if val == "" {
		if p.pos < len(p.tokens) && p.current().Quoted {
			val = strings.TrimSpace(p.consume().Text)
		}
	}
	if val == "" {
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: for filter %q", ErrEmptyFilterValue, key))
	}
	p.FoundFilters[key] = true
	return &FilterExpr{Key: key, Value: val}, nil
}

// parseOperand parses a single operand from the token slice.
// Every operand must be an allowed key:value filter (quoted value may follow key:).
func (p *parser) parseOperand() (QueryExpr, error) {
	if p.pos >= len(p.tokens) {
		return nil, nil
	}

	if p.current().Text == "AND" || p.current().Text == "OR" {
		p.consume()
		return nil, nil
	}
	if p.current().Text == "(" {
		p.consume()

		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}

		if p.pos >= len(p.tokens) || p.current().Text != ")" {
			return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w", ErrExpectedClosingParen))
		}

		p.consume()
		return expr, nil
	}

	cur := p.current()

	if strings.Contains(cur.Text, ":") {
		parts := strings.SplitN(cur.Text, ":", 2)
		key := parts[0]
		val := strings.TrimSpace(parts[1])
		p.consume()
		f, err := p.makeFilter(key, val)
		if err != nil {
			return nil, err
		}
		return f, nil
	}

	if cur.Quoted {
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: use e.g. title:\"...\"", ErrUnexpectedQuotedToken))
	}

	var parts []string
	parts = append(parts, p.consume().Text)
	for p.pos < len(p.tokens) {
		next := p.peek()

		if next.Quoted || next.Text == "(" || next.Text == ")" || next.Text == "AND" || next.Text == "OR" {
			break
		}

		if strings.Contains(next.Text, ":") {
			candidate := strings.ToLower(strings.TrimSpace(strings.SplitN(next.Text, ":", 2)[0]))
			if _, ok := p.allowedKeys[candidate]; ok {
				break
			}
		}

		parts = append(parts, p.consume().Text)
	}

	joined := strings.TrimSpace(strings.Join(parts, " "))
	if joined == "" {
		return nil, nil
	}

	if !strings.Contains(joined, ":") {
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: got %q", ErrExpectedKeyValue, joined))
	}

	parts2 := strings.SplitN(joined, ":", 2)
	key := parts2[0]
	rest := strings.TrimSpace(parts2[1])
	f, err := p.makeFilter(key, rest)
	if err != nil {
		return nil, err
	}
	return f, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseAnd parses a series of operands with an implicit AND between them
func (p *parser) parseAnd() (QueryExpr, error) {
	var children []QueryExpr

	for {
		operand, err := p.parseOperand()
		if err != nil {
			return nil, err
		}
		if operand != nil {
			children = append(children, operand)
		}
		if p.pos >= len(p.tokens) {
			break
		}
		next := p.peek()
		if strings.EqualFold(next.Text, "OR") || next.Text == ")" {
			break
		}
		if strings.EqualFold(next.Text, "AND") {
			p.consume()
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
func (p *parser) parseOr() (QueryExpr, error) {
	expr, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.pos < len(p.tokens) {
		if !strings.EqualFold(p.peek().Text, "OR") {
			break
		}

		p.consume()

		right, err := p.parseAnd()
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

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// assertFullyConsumed checks if the parser has consumed all tokens.
func (p *parser) assertFullyConsumed() error {
	if p.pos >= len(p.tokens) {
		return nil
	}

	t := p.tokens[p.pos]
	return errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: unexpected token %q", ErrTrailingInput, t.Text))
}
