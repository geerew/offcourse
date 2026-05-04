package queryparser

import (
	"errors"
	"fmt"
	"strings"
)

// parser is recursive-descent: expression → OR-chain of AND-groups → operands (filters or parens).
type parser struct {
	tokens       []token
	pos          int
	allowedKeys  map[string]struct{}
	foundFilters map[string]bool
}

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

// parseExpression is the start rule (also used inside "(" ... ")").
func (p *parser) parseExpression() (QueryExpr, error) {
	return p.parseOr()
}

// parseOr parses and ... OR and ...; OR binds loosest.
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

// parseAnd parses one or more operands with implicit AND; stops at OR, ")", or EOF.
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

func (p *parser) current() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}

	return token{Text: "", Quoted: false}
}

func (p *parser) consume() token {
	tok := p.current()
	p.pos++
	return tok
}

func (p *parser) peek() token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}

	return token{Text: "", Quoted: false}
}

// makeFilter normalizes key, checks allowedKeys, resolves value (including quoted token after "key:" alone).
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
	p.foundFilters[key] = true
	return &FilterExpr{Key: key, Value: val}, nil
}

// parseOperand: "(", expression ")"; or key:value (possibly split across tokens); stray AND/OR consumed as empty.
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

		expr, err := p.parseExpression()
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

func (p *parser) assertFullyConsumed() error {
	if p.pos >= len(p.tokens) {
		return nil
	}

	t := p.tokens[p.pos]
	return errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: unexpected token %q", ErrTrailingInput, t.Text))
}
