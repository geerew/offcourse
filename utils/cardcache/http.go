package cardcache

import (
	"strings"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// FormatETag returns a strong ETag for the full card content hash
func FormatETag(cardHash string) string {
	return `"` + cardHash + `"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// MatchIfNoneMatch reports whether ifNoneMatch matches the card hash ETag
func MatchIfNoneMatch(ifNoneMatch, cardHash string) bool {
	if ifNoneMatch == "" || cardHash == "" {
		return false
	}

	want := FormatETag(cardHash)

	for part := range strings.SplitSeq(ifNoneMatch, ",") {
		part = strings.TrimSpace(part)
		if part == "*" || part == want {
			return true
		}

		if strings.HasPrefix(part, "W/") {
			part = strings.TrimSpace(strings.TrimPrefix(part, "W/"))
		}

		if part == want {
			return true
		}
	}

	return false
}
