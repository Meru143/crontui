package crontab

import (
	"strings"
	"testing"

	"github.com/meru143/crontui/pkg/types"
)

func TestFormatCrontab_Basic(t *testing.T) {
	jobs := []types.CronJob{
		{ID: 1, Schedule: "0 * * * *", Command: "/usr/bin/backup", Description: "Hourly backup", Enabled: true},
		{ID: 2, Schedule: "0 0 * * *", Command: "/usr/bin/cleanup", Description: "Nightly cleanup", Enabled: false},
	}

	out := FormatCrontab(jobs)

	// Header present
	if !strings.Contains(out, "# Crontab managed by crontui") {
		t.Error("output should contain header comment")
	}

	// First job annotations
	if !strings.Contains(out, "# description: Hourly backup") {
		t.Error("missing description annotation for job 1")
	}
	// First job is enabled — no leading #
	if !strings.Contains(out, "0 * * * * /usr/bin/backup") {
		t.Error("missing enabled job line for job 1")
	}

	// Second job disabled — line starts with #
	if !strings.Contains(out, "#0 0 * * * /usr/bin/cleanup") {
		t.Error("disabled job should be commented out")
	}
}

func TestFormatCrontab_Annotations(t *testing.T) {
	jobs := []types.CronJob{
		{
			ID:          1,
			Schedule:    "*/5 * * * *",
			Command:     "/usr/bin/task",
			Description: "My task",
			WorkingDir:  "/home/user",
			Mailto:      "dev@example.com",
			Enabled:     true,
		},
	}

	out := FormatCrontab(jobs)
	if !strings.Contains(out, "# description: My task") {
		t.Error("missing description annotation")
	}
	if strings.Contains(out, "# workingdir:") {
		t.Error("workingdir annotation should not be emitted")
	}
	if strings.Contains(out, "# mailto:") {
		t.Error("mailto annotation should not be emitted")
	}
}

func TestFormatCrontab_NoAnnotationsWhenEmpty(t *testing.T) {
	jobs := []types.CronJob{
		{ID: 1, Schedule: "0 * * * *", Command: "/usr/bin/noop", Enabled: true},
	}

	out := FormatCrontab(jobs)
	if strings.Contains(out, "# description:") {
		t.Error("should not emit description annotation when empty")
	}
	if strings.Contains(out, "# workingdir:") {
		t.Error("should not emit workingdir annotation when empty")
	}
	if strings.Contains(out, "# mailto:") {
		t.Error("should not emit mailto annotation when empty")
	}
}

func TestFormatCrontab_EmptyJobs(t *testing.T) {
	out := FormatCrontab(nil)
	if !strings.Contains(out, "# Crontab managed by crontui") {
		t.Error("even empty output should have header")
	}
}

func TestRoundtrip(t *testing.T) {
	original := []types.CronJob{
		{ID: 1, Schedule: "*/5 * * * *", Command: "/usr/bin/task", Description: "My task", Enabled: true, WorkingDir: "/home", Mailto: "x@y.com"},
		{ID: 2, Schedule: "0 0 * * *", Command: "/usr/bin/cleanup", Enabled: false},
	}

	formatted := FormatCrontab(original)
	parsed := ParseCrontab(formatted)

	if len(parsed) != len(original) {
		t.Fatalf("roundtrip: got %d jobs, want %d", len(parsed), len(original))
	}

	for i := range original {
		if parsed[i].Schedule != original[i].Schedule {
			t.Errorf("roundtrip[%d].Schedule = %q, want %q", i, parsed[i].Schedule, original[i].Schedule)
		}
		if parsed[i].Command != original[i].Command {
			t.Errorf("roundtrip[%d].Command = %q, want %q", i, parsed[i].Command, original[i].Command)
		}
		if parsed[i].Description != original[i].Description {
			t.Errorf("roundtrip[%d].Description = %q, want %q", i, parsed[i].Description, original[i].Description)
		}
		if parsed[i].Enabled != original[i].Enabled {
			t.Errorf("roundtrip[%d].Enabled = %v, want %v", i, parsed[i].Enabled, original[i].Enabled)
		}
		if parsed[i].WorkingDir != "" {
			t.Errorf("roundtrip[%d].WorkingDir = %q, want empty string", i, parsed[i].WorkingDir)
		}
		if parsed[i].Mailto != "" {
			t.Errorf("roundtrip[%d].Mailto = %q, want empty string", i, parsed[i].Mailto)
		}
	}
}

func TestWriteDocument_PreservesEnvAssignments(t *testing.T) {
	raw := "SHELL=/bin/bash\nPATH=/usr/local/bin:/usr/bin\n0 * * * * /usr/bin/backup\n"

	doc, err := ParseDocument(raw)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}

	jobs := doc.Jobs()
	jobs[0].Command = "/usr/bin/backup --full"
	if err := doc.ReplaceJobs(jobs); err != nil {
		t.Fatalf("ReplaceJobs: %v", err)
	}

	oldWriteRaw := writeRawCrontabFn
	defer func() {
		writeRawCrontabFn = oldWriteRaw
	}()

	var wrote string
	writeRawCrontabFn = func(content string) error {
		wrote = content
		return nil
	}

	if err := WriteDocument(doc); err != nil {
		t.Fatalf("WriteDocument: %v", err)
	}

	for _, want := range []string{
		"SHELL=/bin/bash",
		"PATH=/usr/local/bin:/usr/bin",
		"0 * * * * /usr/bin/backup --full",
	} {
		if !strings.Contains(wrote, want) {
			t.Fatalf("WriteDocument output missing %q\nfull output:\n%s", want, wrote)
		}
	}
}
