# Releasing CronTUI

CronTUI publishes releases from semver tags (`v1.0.1`, `v1.0.2`, and so on). `go install github.com/meru143/crontui@latest` resolves to the newest tagged version, not the newest commit on `master`.

## Before You Tag

Make sure the release commit is already on `master` and that verification is green:

```bash
go test ./...
go vet ./...
golangci-lint run --timeout 5m
```

For native Windows support changes, also verify the real Task Scheduler smoke:

```powershell
go build -o crontui.exe .
.\scripts\windows-smoke.ps1 -BinaryPath "$PWD\crontui.exe"
```

## Create A Release Tag

1. Open the repository on GitHub.
2. Go to `Actions`.
3. Open `Manual Release Tag`.
4. Click `Run workflow`.
5. Choose `patch`, `minor`, or `major`.

That workflow checks out `master`, computes the next semver tag, creates an annotated tag, and pushes it.

## What Happens Next

- The pushed `v*` tag triggers the `Release` workflow.
- GoReleaser builds the release artifacts and publishes the GitHub release.
- `go install github.com/meru143/crontui@latest` starts resolving to the new tag.
- CI now covers both `ubuntu-latest` and `windows-latest`, including the native Windows Task Scheduler smoke script.

## Install Paths

- Stable tagged release:
  - `go install github.com/meru143/crontui@latest`
- Latest branch tip:
  - `go install github.com/meru143/crontui@master`
- Release binaries:
  - Download from [GitHub Releases](https://github.com/meru143/crontui/releases)
