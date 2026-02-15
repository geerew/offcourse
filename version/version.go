package version

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// The following variables are set via ldflags during a build
var (
	// Version of the build (dev or a release tag)
	Version = "dev"

	// Commit hash of the build
	Commit = "unknown"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetVersion returns a version string
func GetVersion() string {
	// When the version is empty or dev return 'dev-commit' or just 'dev' if the
	// commit is unknown
	if Version == "" || Version == "dev" {
		if Commit != "unknown" && Commit != "" {
			return "dev-" + Commit
		}

		return "dev"
	}

	return Version
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// GetCommit returns the commit hash
func GetCommit() string {
	return Commit
}
