// Package version holds build-time version information injected via ldflags.
package version

import "fmt"

// These variables are set at build time via -ldflags.
//
//	go build -ldflags "-X github.com/meru143/crontui/internal/version.Version=v1.0.0
//	                    -X github.com/meru143/crontui/internal/version.Commit=abc1234"
var (
	Version = "dev"
	Commit  = "none"
)

// Full returns a formatted version string including commit hash.
func Full() string {
	return fmt.Sprintf("crontui %s (commit: %s)", Version, Commit)
}
