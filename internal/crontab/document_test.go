package crontab

import (
	"strings"
	"testing"
)

func TestParseDocument_PreservesEnvAndComments(t *testing.T) {
	raw := strings.Join([]string{
		"# existing header",
		"SHELL=/bin/bash",
		"PATH=/usr/local/bin:/usr/bin",
		"",
		"# description: backup",
		"0 * * * * /usr/bin/backup",
		"# trailing comment",
		"",
	}, "\n")

	doc, err := ParseDocument(raw)
	if err != nil {
		t.Fatalf("ParseDocument returned error: %v", err)
	}

	if got := doc.Render(); got != raw {
		t.Fatalf("Render mismatch\n--- got ---\n%s\n--- want ---\n%s", got, raw)
	}
}

func TestParseDocument_PreservesDisabledDescriptorJobs(t *testing.T) {
	raw := strings.Join([]string{
		"#@reboot /usr/bin/startup",
		"# @hourly /usr/bin/hourly",
		"",
	}, "\n")

	doc, err := ParseDocument(raw)
	if err != nil {
		t.Fatalf("ParseDocument returned error: %v", err)
	}

	jobs := doc.Jobs()
	if len(jobs) != 2 {
		t.Fatalf("Jobs length = %d, want 2", len(jobs))
	}
	if jobs[0].Schedule != "@reboot" || jobs[0].Enabled {
		t.Fatalf("job[0] = %+v, want disabled @reboot", jobs[0])
	}
	if jobs[1].Schedule != "@hourly" || jobs[1].Enabled {
		t.Fatalf("job[1] = %+v, want disabled @hourly", jobs[1])
	}

	if got := doc.Render(); got != raw {
		t.Fatalf("Render mismatch\n--- got ---\n%s\n--- want ---\n%s", got, raw)
	}
}

func TestDocumentRender_PreservesUnknownLinesAfterJobMutation(t *testing.T) {
	raw := strings.Join([]string{
		"SHELL=/bin/bash",
		"# keep this comment",
		"0 * * * * /usr/bin/backup",
		"MAILTO=admin@example.com",
		"15 4 * * * /usr/bin/report",
		"",
	}, "\n")

	doc, err := ParseDocument(raw)
	if err != nil {
		t.Fatalf("ParseDocument returned error: %v", err)
	}

	jobs := doc.Jobs()
	if len(jobs) != 2 {
		t.Fatalf("Jobs length = %d, want 2", len(jobs))
	}

	jobs[0].Command = "/usr/bin/backup --full"
	if err := doc.ReplaceJobs(jobs); err != nil {
		t.Fatalf("ReplaceJobs returned error: %v", err)
	}

	got := doc.Render()
	for _, wantLine := range []string{
		"SHELL=/bin/bash",
		"# keep this comment",
		"MAILTO=admin@example.com",
		"15 4 * * * /usr/bin/report",
		"0 * * * * /usr/bin/backup --full",
	} {
		if !strings.Contains(got, wantLine) {
			t.Fatalf("Render missing %q\nfull output:\n%s", wantLine, got)
		}
	}
}
