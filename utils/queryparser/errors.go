package queryparser

import "errors"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

var (
	ErrInvalidSyntax = errors.New("invalid query syntax")

	ErrUnknownFilterKey      = errors.New("unknown filter key")
	ErrEmptyFilterValue      = errors.New("empty filter value")
	ErrExpectedClosingParen  = errors.New("expected closing ')'")
	ErrUnexpectedQuotedToken = errors.New("unexpected quoted token")
	ErrExpectedKeyValue      = errors.New("expected key:value filter")
	ErrTrailingInput         = errors.New("trailing input after complete query")
	ErrUnterminatedQuote     = errors.New("unterminated quoted string")
)
