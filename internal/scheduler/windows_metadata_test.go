package scheduler

import "testing"

func TestTaskNameForID_RoundTrip(t *testing.T) {
	name := taskNameForID(42)
	if name != "job-42" {
		t.Fatalf("taskNameForID(42) = %q, want %q", name, "job-42")
	}

	id, err := parseIDFromTaskName(name)
	if err != nil {
		t.Fatalf("parseIDFromTaskName returned error: %v", err)
	}
	if id != 42 {
		t.Fatalf("parseIDFromTaskName(%q) = %d, want %d", name, id, 42)
	}
}

func TestParseIDFromTaskName_RejectsInvalidNames(t *testing.T) {
	for _, name := range []string{"", "job-", "job-abc", "cron-42", "job--1"} {
		if _, err := parseIDFromTaskName(name); err == nil {
			t.Fatalf("parseIDFromTaskName(%q) should fail", name)
		}
	}
}

func TestTaskSource_RoundTrip(t *testing.T) {
	source := encodeTaskSource("0 9 * * 1-5")

	schedule, err := decodeTaskSource(source)
	if err != nil {
		t.Fatalf("decodeTaskSource returned error: %v", err)
	}
	if schedule != "0 9 * * 1-5" {
		t.Fatalf("decodeTaskSource(%q) = %q, want %q", source, schedule, "0 9 * * 1-5")
	}
}

func TestDecodeTaskSource_RejectsInvalidMetadata(t *testing.T) {
	for _, source := range []string{
		"",
		"crontui:v1",
		"schedule=MC45ICogKiAxLTU=",
		"crontui:v1;schedule=!!!",
	} {
		if _, err := decodeTaskSource(source); err == nil {
			t.Fatalf("decodeTaskSource(%q) should fail", source)
		}
	}
}
