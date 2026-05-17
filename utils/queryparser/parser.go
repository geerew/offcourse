package queryparser

import (
	"errors"
	"fmt"
	"strings"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parser represents a parsed query
type parser struct {
	tokens       []token
	pos          int
	allowedKeys  map[string]struct{}
	foundFilters map[string]bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// newParser creates a new parser with the given tokens and allowed keys
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
		foundFilters: found,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseOr parses an OR expression
func (p *parser) parseOr() (QueryExpr, error) {
	expr, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.pos < len(p.tokens) {
		if p.current().kind != tokOr {
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

// parseAnd parses an AND expression
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

		next := p.current()

		if next.kind == tokOr || next.kind == tokRParen {
			break
		}

		if next.kind == tokAnd {
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

// current returns the token at pos, or tokInvalid if past the end.
func (p *parser) current() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}

	return token{kind: tokInvalid}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// consume advances pos and returns the token that was at pos before advancing.
func (p *parser) consume() token {
	tok := p.current()
	p.pos++
	return tok
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// makeFilter creates a new FilterExpr with the given key and value, if the key is allowed
func (p *parser) makeFilter(key, val string) (*FilterExpr, error) {
	key = strings.ToLower(strings.TrimSpace(key))
	if _, ok := p.allowedKeys[key]; !ok {
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: %q", ErrUnknownFilterKey, key))
	}

	val = strings.TrimSpace(val)
	if val == "" {
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: for filter %q", ErrEmptyFilterValue, key))
	}

	p.foundFilters[key] = true

	return &FilterExpr{Key: key, Value: val}, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseOperand parses an operand (filter or parentheses)
func (p *parser) parseOperand() (QueryExpr, error) {
	if p.pos >= len(p.tokens) {
		return nil, nil
	}

	switch p.current().kind {
	case tokAnd, tokOr:
		p.consume()
		return nil, nil
	case tokLParen:
		p.consume()

		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}

		if p.pos >= len(p.tokens) || p.current().kind != tokRParen {
			return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w", ErrExpectedClosingParen))
		}

		p.consume()
		return expr, nil
	case tokFilter:
		t := p.consume()
		return p.makeFilter(t.key, t.value)
	case tokLiteral:
		t := p.consume()
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: got %q", ErrExpectedKeyValue, t.raw))
	default:
		t := p.consume()
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: unexpected token %q", ErrTrailingInput, t.String()))
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// assertFullyConsumed checks if the parser has consumed all tokens
func (p *parser) assertFullyConsumed() error {
	if p.pos >= len(p.tokens) {
		return nil
	}

	return errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: unexpected token %q", ErrTrailingInput, p.tokens[p.pos].String()))
}
