package utils

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"os"
	"runtime"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NormalizeWindowsDrive normalizes a given path to ensure that drive letters on Windows
// are correctly interpreted. If the path starts with a drive letter, it appends a
// backslash (\) to paths like "C:" to make them "C:\", and inserts a backslash in paths
// like "C:folder" to make them "C:\folder"
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

// GetEnvOr returns the value of the environment variable if set, otherwise the default
func GetEnvOr(env string, def string) string {
	out := os.Getenv(env)
	if out == "" {
		return def
	}

	return out
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DecodeString is a function that receives a Base64-encoded string and first decodes
// it from Base64 and then URL-decodes it. The function returns the decoded string, or
// an error if either of the decoding operations fails. It uses standard library
// functions for both decoding operations.
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

// EncodeString is a function that receives a string, URL-encodes it, and then encodes
// the result in Base64. The function returns the Base64-encoded string. It uses
// standard library functions for both encoding operations.
func EncodeString(p string) string {
	encodedPath := url.QueryEscape(p)

	res := base64.StdEncoding.EncodeToString([]byte(encodedPath))

	return res
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Map is a generic function that takes a slice of type T and a function that
// maps T to type V. It returns a new slice of type V with the mapped values
func Map[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))

	for i, t := range ts {
		result[i] = fn(t)
	}

	return result
}
