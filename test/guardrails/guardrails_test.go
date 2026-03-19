package guardrails

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCIWorkflowTargetsMasterBranch(t *testing.T) {
	content, err := os.ReadFile(repoPath(t, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("read ci workflow: %v", err)
	}

	workflow := string(content)

	if strings.Contains(workflow, "branches: [main]") {
		t.Fatalf("ci workflow still targets main branch")
	}

	if count := strings.Count(workflow, "branches: [master]"); count != 2 {
		t.Fatalf("expected ci workflow to target master for push and pull_request, found %d occurrences", count)
	}

	required := []string{
		"actions/checkout@v6",
		"actions/setup-go@v6",
		"golangci/golangci-lint-action@v9",
		"ubuntu-latest",
		"windows-latest",
		"runner.os == 'Windows'",
		"scripts\\windows-smoke.ps1",
		"CRONTUI_WINDOWS_TASK_PATH",
	}

	for _, needle := range required {
		if !strings.Contains(workflow, needle) {
			t.Fatalf("ci workflow missing %q", needle)
		}
	}
}

func TestGoReleaserConfigDoesNotReferenceMissingHomebrewTap(t *testing.T) {
	content, err := os.ReadFile(repoPath(t, ".goreleaser.yaml"))
	if err != nil {
		t.Fatalf("read goreleaser config: %v", err)
	}

	config := string(content)

	if strings.Contains(config, "homebrew-tap") {
		t.Fatal("goreleaser config still references missing homebrew tap repository")
	}

	if strings.Contains(config, "\nbrews:") {
		t.Fatal("goreleaser config still enables Homebrew publishing")
	}
}

func TestManualReleaseWorkflowSupportsSemverBumps(t *testing.T) {
	content, err := os.ReadFile(repoPath(t, ".github", "workflows", "manual-release.yml"))
	if err != nil {
		t.Fatalf("read manual release workflow: %v", err)
	}

	workflow := string(content)

	required := []string{
		"name: Manual Release Tag",
		"workflow_dispatch:",
		"bump:",
		"type: choice",
		"- patch",
		"- minor",
		"- major",
		"contents: write",
		"actions/checkout@v6",
		"ref: refs/heads/master",
		"git tag -a",
		"git push origin",
	}

	for _, needle := range required {
		if !strings.Contains(workflow, needle) {
			t.Fatalf("manual release workflow missing %q", needle)
		}
	}
}

func TestWindowsSmokeScriptExitsExplicitlyOnSuccess(t *testing.T) {
	data, err := os.ReadFile(repoPath(t, "scripts", "windows-smoke.ps1"))
	if err != nil {
		t.Fatalf("ReadFile(windows-smoke.ps1): %v", err)
	}

	script := string(data)
	if !strings.Contains(script, "exit 0") {
		t.Fatalf("windows smoke script should exit 0 explicitly after successful validation")
	}
}

func TestReleaseWorkflowUploadsDemoAssets(t *testing.T) {
	content, err := os.ReadFile(repoPath(t, ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	workflow := string(content)
	required := []string{
		"actions/checkout@v6",
		"actions/setup-go@v6",
		"goreleaser/goreleaser-action@v7",
		"gh release upload",
		"media/demo/*.gif",
		"--clobber",
	}

	for _, needle := range required {
		if !strings.Contains(workflow, needle) {
			t.Fatalf("release workflow missing %q", needle)
		}
	}
}

func repoPath(t *testing.T, elems ...string) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve guardrail test file path")
	}

	parts := append([]string{filepath.Dir(file), "..", ".."}, elems...)
	return filepath.Clean(filepath.Join(parts...))
}
