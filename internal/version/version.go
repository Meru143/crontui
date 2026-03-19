// Package version holds build-time version information injected via ldflags.
package version

import (
	"fmt"
	"runtime/debug"
)

// These variables are set at build time via -ldflags.
//
//	go build -ldflags "-X github.com/meru143/crontui/internal/version.Version=v1.0.0
//	                    -X github.com/meru143/crontui/internal/version.Commit=abc1234"
var (
	Version = "dev"
	Commit  = "none"

	readBuildInfo = debug.ReadBuildInfo
)

// Full returns a formatted version string including commit hash.
func Full() string {
	version := resolvedVersion()
	commit := resolvedCommit()

	if commit == "" {
		return fmt.Sprintf("crontui %s", version)
	}

	return fmt.Sprintf("crontui %s (commit: %s)", version, commit)
}

func resolvedVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}

	info, ok := readBuildInfo()
	if ok && info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}

	return "dev"
}

func resolvedCommit() string {
	if Commit != "" && Commit != "none" {
		return Commit
	}

	info, ok := readBuildInfo()
	if !ok || info == nil {
		return ""
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			return shortCommit(setting.Value)
		}
	}

	return ""
}

func shortCommit(revision string) string {
	if len(revision) <= 7 {
		return revision
	}
	return revision[:7]
}
