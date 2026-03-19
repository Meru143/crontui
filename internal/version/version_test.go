package version

import (
	"runtime/debug"
	"testing"
)

func TestFull_PrefersInjectedLdflags(t *testing.T) {
	oldVersion := Version
	oldCommit := Commit
	oldReadBuildInfo := readBuildInfo
	defer func() {
		Version = oldVersion
		Commit = oldCommit
		readBuildInfo = oldReadBuildInfo
	}()

	Version = "v1.1.0"
	Commit = "abc1234"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/meru143/crontui",
				Version: "v9.9.9",
			},
		}, true
	}

	if got := Full(); got != "crontui v1.1.0 (commit: abc1234)" {
		t.Fatalf("Full() = %q, want %q", got, "crontui v1.1.0 (commit: abc1234)")
	}
}

func TestFull_FallsBackToModuleVersionWithoutCommit(t *testing.T) {
	oldVersion := Version
	oldCommit := Commit
	oldReadBuildInfo := readBuildInfo
	defer func() {
		Version = oldVersion
		Commit = oldCommit
		readBuildInfo = oldReadBuildInfo
	}()

	Version = "dev"
	Commit = "none"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/meru143/crontui",
				Version: "v1.1.0",
			},
		}, true
	}

	if got := Full(); got != "crontui v1.1.0" {
		t.Fatalf("Full() = %q, want %q", got, "crontui v1.1.0")
	}
}

func TestFull_FallsBackToBuildInfoCommit(t *testing.T) {
	oldVersion := Version
	oldCommit := Commit
	oldReadBuildInfo := readBuildInfo
	defer func() {
		Version = oldVersion
		Commit = oldCommit
		readBuildInfo = oldReadBuildInfo
	}()

	Version = "dev"
	Commit = "none"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{
				Path:    "github.com/meru143/crontui",
				Version: "(devel)",
			},
			Settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "1234567890abcdef"},
			},
		}, true
	}

	if got := Full(); got != "crontui dev (commit: 1234567)" {
		t.Fatalf("Full() = %q, want %q", got, "crontui dev (commit: 1234567)")
	}
}

func TestFull_DevBuildWithoutMetadata(t *testing.T) {
	oldVersion := Version
	oldCommit := Commit
	oldReadBuildInfo := readBuildInfo
	defer func() {
		Version = oldVersion
		Commit = oldCommit
		readBuildInfo = oldReadBuildInfo
	}()

	Version = "dev"
	Commit = "none"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}

	if got := Full(); got != "crontui dev" {
		t.Fatalf("Full() = %q, want %q", got, "crontui dev")
	}
}
