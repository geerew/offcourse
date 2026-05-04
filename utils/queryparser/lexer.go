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

// Token represents a token with its text and whether it was quoted
type token struct {
	Text   string
	Quoted bool
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// tokenize tokenizes an input string while respecting quoted substrings
//
// Unterminated quoted input returns an error
func tokenize(input string) ([]token, error) {
	var tokens []token
	var current strings.Builder
	var q quoteMode

	// flushUnquoted flushes the current unquoted token to the tokens slice
	flushUnquoted := func() {
		if current.Len() > 0 {
			tokens = append(tokens, token{Text: current.String(), Quoted: false})
			current.Reset()
		}
	}

	// singleQuoteCanOpen checks if a single quote can open a quoted region
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
			// Handle unquoted tokens
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
