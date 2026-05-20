package cardcache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully formatting an ETag
func TestFormatETag(t *testing.T) {
	require.Equal(t, `"abc"`, FormatETag("abc"))
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully matching an ETag
func TestMatchIfNoneMatch(t *testing.T) {
	hash := "deadbeef"
	etag := FormatETag(hash)

	require.False(t, MatchIfNoneMatch("", hash))
	require.False(t, MatchIfNoneMatch(etag, ""))
	require.True(t, MatchIfNoneMatch(etag, hash))
	require.True(t, MatchIfNoneMatch(etag+", "+etag, hash))
	require.True(t, MatchIfNoneMatch("W/"+etag, hash))
	require.True(t, MatchIfNoneMatch("*", hash))
	require.False(t, MatchIfNoneMatch(`"other"`, hash))
}
