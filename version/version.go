package version

import "strings"

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// These variables are set via ldflags during the build
var (
	// Version is the git tag (e.g., "v0.0.1") or "dev" for dev builds
	Version = "dev"

	// Commit is the short git commit hash
	Commit = "unknown"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetVersion returns the version string
// For dev builds (when Version is "dev" or empty), it returns "dev-[commit-hash]"
// For release builds, it returns the git tag
func GetVersion() string {
	// If version is "dev" and we have a commit hash, return "dev-{commit}"
	if Version == "dev" {
		if Commit != "unknown" && Commit != "" {
			return "dev-" + Commit
		}
		return "dev"
	}
	// If version is empty, try to use commit hash
	if Version == "" {
		if Commit != "unknown" && Commit != "" {
			return "dev-" + Commit
		}
		return "dev"
	}
	// Otherwise return the version (which should be a tag)
	return Version
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetCommit returns the commit hash
func GetCommit() string {
	return Commit
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// IsDev returns true if this is a dev build
func IsDev() bool {
	return Version == "dev" || strings.HasPrefix(Version, "dev-")
}
