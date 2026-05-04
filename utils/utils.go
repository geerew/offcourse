package utils

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NormalizeWindowsDrive normalizes a given paths that start with a drive letter (e.g. "C:")
// are correctly interpreted on Windows systems, by appending a backslash (\)
//
// For example, "C:" becomes "C:\" and "C:folder" becomes "C:\folder"
//
// Skipped on non-Windows platforms
func NormalizeWindowsDrive(path string) string {
	if runtime.GOOS == "windows" {
		if len(path) >= 2 && path[1] == ':' {
			if len(path) == 2 {
				path += `\`
			} else if path[2] != '\\' {
				path = path[:2] + `\` + path[2:]
			}
		}
	}

	return path
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetEnvOr returns the value of the environment variable if set, otherwise returns the
// default
func GetEnvOr(env string, def string) string {
	out := os.Getenv(env)
	if out == "" {
		return def
	}

	return out
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DecodeString decodes a base64 encoded string
func DecodeString(p string) (string, error) {
	bytePath, err := base64.StdEncoding.DecodeString(p)
	if err != nil {
		return "", fmt.Errorf("failed to decode path")
	}

	decodedPath, err := url.QueryUnescape(string(bytePath))
	if err != nil {
		return "", fmt.Errorf("failed to unescape path")

	}

	return decodedPath, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// EncodeString encodes a string into base64
func EncodeString(p string) string {
	encodedPath := url.QueryEscape(p)
	res := base64.StdEncoding.EncodeToString([]byte(encodedPath))
	return res
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Map runs a function over a slice of type T, returning a new slice of type V
func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))

	for i, t := range ts {
		result[i] = fn(t)
	}

	return result
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsCard returns true when the filename is `card.[valid-ext]`
func IsCard(filename string) bool {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))

	if ext == "" {
		return false
	}

	name := strings.TrimSuffix(filename, "."+ext)
	if name != "card" {
		return false
	}

	switch ext {
	case "jpg", "jpeg", "png", "webp", "tiff":
		return true
	default:
		return false
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// StringSplit splits a string into a slice of strings, trimming each string and removing
// empty strings
func StringSplit(s string, sep string) []string {
	var out []string

	for _, p := range strings.Split(s, sep) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}

	return out
}
