package queryparser

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type quoteMode byte

const (
	qNone quoteMode = iota
	qDouble
	qSingle
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Token represents a lexeme after splitting on whitespace and delimiters
type token struct {
	Text   string
	Quoted bool
}

// tokenize splits input on whitespace and parentheses; builds quoted tokens for strings in "..."
// or '...' when ' starts a value (empty token, or immediately after ':').
func tokenize(input string) ([]token, error) {
	var tokens []token
	var current strings.Builder
	var q quoteMode

	flushUnquoted := func() {
		if current.Len() > 0 {
			tokens = append(tokens, token{Text: current.String(), Quoted: false})
			current.Reset()
		}
	}

	// Mid-word apostrophe stays literal (e.g. don't); leading ' or ':'+' opens a quoted value.
	singleQuoteCanOpen := func() bool {
		s := current.String()
		return len(s) == 0 || strings.HasSuffix(s, ":")
	}

	for _, r := range input {
		switch q {
		case qDouble:
			if r == '"' {
				tokens = append(tokens, token{Text: current.String(), Quoted: true})
				current.Reset()
				q = qNone
				continue
			}
			current.WriteRune(r)

		case qSingle:
			if r == '\'' {
				tokens = append(tokens, token{Text: current.String(), Quoted: true})
				current.Reset()
				q = qNone
				continue
			}
			current.WriteRune(r)

		default:
			switch {
			case r == '"':
				flushUnquoted()
				q = qDouble

			case r == '\'':
				if singleQuoteCanOpen() {
					flushUnquoted()
					q = qSingle
				} else {
					current.WriteRune(r)
				}

			case unicode.IsSpace(r):
				flushUnquoted()

			case r == '(' || r == ')':
				flushUnquoted()
				tokens = append(tokens, token{Text: string(r), Quoted: false})

			default:
				current.WriteRune(r)
			}
		}
	}

	if q != qNone {
		return nil, errors.Join(ErrInvalidSyntax, fmt.Errorf("%w: unterminated quoted string", ErrUnterminatedQuote))
	}

	if current.Len() > 0 {
		tokens = append(tokens, token{Text: current.String(), Quoted: false})
	}

	return tokens, nil
}
