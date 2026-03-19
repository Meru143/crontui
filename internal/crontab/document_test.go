package crontab

import (
	"strings"
	"testing"

	"github.com/meru143/crontui/pkg/types"
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

func TestParseDocument_PreservesManagedIDs(t *testing.T) {
	raw := strings.Join([]string{
		"# crontui:id: 7",
		"0 * * * * /usr/bin/a",
		"# crontui:id: 42",
		"#30 2 * * * /usr/bin/b",
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
	if jobs[0].ID != 7 {
		t.Fatalf("job[0].ID = %d, want 7", jobs[0].ID)
	}
	if jobs[1].ID != 42 {
		t.Fatalf("job[1].ID = %d, want 42", jobs[1].ID)
	}
	if jobs[1].Enabled {
		t.Fatalf("job[1] should be disabled: %+v", jobs[1])
	}

	if got := doc.Render(); got != raw {
		t.Fatalf("Render mismatch\n--- got ---\n%s\n--- want ---\n%s", got, raw)
	}
}

func TestDocumentReplaceJobs_BackfillsManagedIDsForLegacyJobs(t *testing.T) {
	raw := strings.Join([]string{
		"0 * * * * /usr/bin/a",
		"15 4 * * * /usr/bin/b",
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

	if err := doc.ReplaceJobs(jobs); err != nil {
		t.Fatalf("ReplaceJobs returned error: %v", err)
	}

	rendered := doc.Render()
	for _, wantLine := range []string{
		"# crontui:id: 1",
		"0 * * * * /usr/bin/a",
		"# crontui:id: 2",
		"15 4 * * * /usr/bin/b",
	} {
		if !strings.Contains(rendered, wantLine) {
			t.Fatalf("Render missing %q\nfull output:\n%s", wantLine, rendered)
		}
	}
}

func TestDocumentReplaceJobs_KeepsManagedIDsStableAcrossDeleteAndAdd(t *testing.T) {
	raw := strings.Join([]string{
		"# crontui:id: 1",
		"0 * * * * /usr/bin/a",
		"# crontui:id: 2",
		"15 4 * * * /usr/bin/b",
		"# crontui:id: 3",
		"30 6 * * * /usr/bin/c",
		"",
	}, "\n")

	doc, err := ParseDocument(raw)
	if err != nil {
		t.Fatalf("ParseDocument returned error: %v", err)
	}

	jobs := doc.Jobs()
	if len(jobs) != 3 {
		t.Fatalf("Jobs length = %d, want 3", len(jobs))
	}

	jobs = []types.CronJob{
		jobs[0],
		jobs[2],
		{ID: 4, Schedule: "@reboot", Command: "/usr/bin/d", Enabled: true},
	}

	if err := doc.ReplaceJobs(jobs); err != nil {
		t.Fatalf("ReplaceJobs returned error: %v", err)
	}

	roundTrip, err := ParseDocument(doc.Render())
	if err != nil {
		t.Fatalf("ParseDocument(roundTrip) returned error: %v", err)
	}

	gotJobs := roundTrip.Jobs()
	if len(gotJobs) != 3 {
		t.Fatalf("roundTrip Jobs length = %d, want 3", len(gotJobs))
	}

	wantIDs := []int{1, 3, 4}
	for i, wantID := range wantIDs {
		if gotJobs[i].ID != wantID {
			t.Fatalf("job[%d].ID = %d, want %d", i, gotJobs[i].ID, wantID)
		}
	}
}
