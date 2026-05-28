package queryparser

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type tokenKind byte

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

const (
	tokInvalid tokenKind = iota
	tokLParen
	tokRParen
	tokAnd
	tokOr
	tokFilter
	tokLiteral // unquoted chunk with no ':' (parser rejects)
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// token is one lexeme from the lexer
type token struct {
	kind  tokenKind
	key   string // tokFilter
	value string // tokFilter
	raw   string // tokLiteral (display / errors)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// String returns a string representation of the token
func (t token) String() string {
	switch t.kind {
	case tokLParen:
		return "("
	case tokRParen:
		return ")"
	case tokAnd:
		return "AND"
	case tokOr:
		return "OR"
	case tokFilter:
		return t.key + ":" + t.value
	case tokLiteral:
		return t.raw
	default:
		return ""
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// tokenize converts an input string into a slice of tokens
func tokenize(input string) ([]token, error) {
	var out []token
	i := 0

	for {
		i = skipSpace(input, i)

		if i >= len(input) {
			return out, nil
		}

		switch input[i] {
		case '(':
			out = append(out, token{kind: tokLParen})
			i++
			continue
		case ')':
			out = append(out, token{kind: tokRParen})
			i++
			continue
		}

		if isKeyword(input, i, "and") {
			out = append(out, token{kind: tokAnd})
			i += len("and")
			continue
		}

		if isKeyword(input, i, "or") {
			out = append(out, token{kind: tokOr})
			i += len("or")
			continue
		}

		if input[i] == '"' || input[i] == '\'' {
			return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: use key:\"value\" after filter key", ErrUnexpectedQuotedToken))
		}

		tok, ni, err := lexFilterToken(input, i)
		if err != nil {
			return nil, err
		}

		out = append(out, tok)
		i = ni
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// skipSpace skips whitespace in the input string
func skipSpace(s string, i int) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsSpace(r) {
			break
		}

		i += size
	}

	return i
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isKeyword checks if the input string is a keyword
func isKeyword(s string, i int, kw string) bool {
	n := len(kw)

	if i+n > len(s) {
		return false
	}

	return strings.EqualFold(s[i:i+n], kw) && endBoundary(s, i+n)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// endBoundary checks if the next rune may not continue an identifier or start a filter (':')
func endBoundary(s string, i int) bool {
	if i >= len(s) {
		return true
	}

	r, _ := utf8.DecodeRuneInString(s[i:])
	if r == ':' {
		return false
	}

	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
		return false
	}

	return true
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isKeyRune checks if the rune is a valid character for a key
func isKeyRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// lexFilterToken returns tokFilter (key + value) or tokLiteral if there is no ':' in this chunk.
func lexFilterToken(s string, i int) (token, int, error) {
	start := i
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == ':' {
			break
		}

		if unicode.IsSpace(r) || r == '(' || r == ')' {
			return token{kind: tokLiteral, raw: s[start:i]}, i, nil
		}

		if !isKeyRune(r) {
			return token{}, i, fmt.Errorf("%w: invalid character in filter key", ErrInvalidSyntax)
		}

		i += size
	}

	if i >= len(s) || s[i] != ':' {
		return token{kind: tokLiteral, raw: s[start:i]}, i, nil
	}

	key := s[start:i]
	if key == "" {
		return token{}, i, errors.Join(ErrInvalidSyntax, fmt.Errorf("empty filter key before ':'"))
	}

	i++

	var val string
	var err error
	switch {
	case i >= len(s):
		return token{}, i, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: missing value after ':'", ErrEmptyFilterValue))
	case s[i] == '"':
		val, i, err = readQuoted(s, i+1, '"')
		if err != nil {
			return token{}, i, err
		}
	case s[i] == '\'':
		val, i, err = readQuoted(s, i+1, '\'')
		if err != nil {
			return token{}, i, err
		}
	default:
		val, i = readUnquotedValue(s, i)
	}

	return token{kind: tokFilter, key: key, value: val}, i, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// readQuoted reads a quoted value from the input string
func readQuoted(s string, i int, quote byte) (string, int, error) {
	var b strings.Builder
	for i < len(s) {
		c := s[i]
		if c == quote {
			return b.String(), i + 1, nil
		}

		if c == '\\' && i+1 < len(s) {
			n := s[i+1]
			if n == '\\' || n == quote {
				b.WriteByte(n)
				i += 2
				continue
			}
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		b.WriteRune(r)
		i += size
	}

	return "", i, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w", ErrUnterminatedQuote))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// readUnquotedValue reads an unquoted value from the input string
func readUnquotedValue(s string, i int) (string, int) {
	start := i
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if unicode.IsSpace(r) || r == '(' || r == ')' {
			break
		}

		i += size
	}

	return s[start:i], i
}
