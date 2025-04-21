package version

import "runtime"

// These variables are set by ldflags during build.
var (
	version   = "dev"     // App version (e.g., v1.0.0)
	buildDate = "unknown" // Build date (RFC3339)
	gitCommit = "unknown" // Git commit SHA
)

// BuildInfo contains version and build details.
type BuildInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"buildDate"`
	GitCommit string `json:"gitCommit"`
	GoVersion string `json:"goVersion"`
}

// Get returns the build information.
func Get() BuildInfo {
	return BuildInfo{
		Version:   version,
		BuildDate: buildDate,
		GitCommit: gitCommit,
		GoVersion: runtime.Version(), // Get Go version from runtime
	}
}
