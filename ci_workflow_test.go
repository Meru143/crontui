package main

import (
	"os"
	"strings"
	"testing"
)

func TestCIWorkflowTargetsMasterBranch(t *testing.T) {
	content, err := os.ReadFile(".github/workflows/ci.yml")
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
