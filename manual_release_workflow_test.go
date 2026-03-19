package main

import (
	"os"
	"strings"
	"testing"
)

func TestManualReleaseWorkflowSupportsSemverBumps(t *testing.T) {
	content, err := os.ReadFile(".github/workflows/manual-release.yml")
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
