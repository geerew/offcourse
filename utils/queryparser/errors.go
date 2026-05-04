package queryparser

import "errors"

// ErrInvalidSyntax wraps parser rejections (free text, unknown filter key, etc.).
var ErrInvalidSyntax = errors.New("invalid query syntax")
