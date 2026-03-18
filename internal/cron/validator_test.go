package cron

import "testing"

func TestValidate(t *testing.T) {
	valid := []string{
		"* * * * *",
		"0 * * * *",
		"0 0 * * *",
		"0 0 * * 0",
		"0 0 1 * *",
		"0 0 1 1 *",
		"*/5 * * * *",
		"0 9-17 * * 1-5",
		"@hourly",
		"@daily",
		"@weekly",
		"@monthly",
		"@yearly",
		"@annually",
	}
	for _, expr := range valid {
		ok, err := Validate(expr)
		if !ok || err != nil {
			t.Errorf("Validate(%q) = (%v, %v), want (true, nil)", expr, ok, err)
		}
	}

	invalid := []string{
		"",
		"not-a-cron",
		"* * *",
		"60 * * * *",
		"* 25 * * *",
		"* * 32 * *",
		"* * * 13 *",
		"* * * * 8",
	}
	for _, expr := range invalid {
		ok, err := Validate(expr)
		if ok && err == nil {
			t.Errorf("Validate(%q) should have failed", expr)
		}
	}
}

func TestPreview(t *testing.T) {
	valid, runs, _, _ := Preview("0 * * * *", 3)
	if !valid {
		t.Error("Preview(\"0 * * * *\") should be valid")
	}
	if len(runs) != 3 {
		t.Errorf("expected 3 next runs, got %d", len(runs))
	}

	valid2, _, _, errMsg := Preview("invalid", 3)
	if valid2 {
		t.Error("Preview(\"invalid\") should not be valid")
	}
	if errMsg == "" {
		t.Error("Preview(\"invalid\") should return error message")
	}
}

func TestValidate_FieldRanges(t *testing.T) {
	tests := []struct {
		expr  string
		valid bool
	}{
		// Minute: 0-59
		{"0 * * * *", true},
		{"59 * * * *", true},
		{"60 * * * *", false},
		// Hour: 0-23
		{"* 0 * * *", true},
		{"* 23 * * *", true},
		{"* 24 * * *", false},
		// Day of month: 1-31
		{"* * 1 * *", true},
		{"* * 31 * *", true},
		{"* * 0 * *", false},
		{"* * 32 * *", false},
		// Month: 1-12
		{"* * * 1 *", true},
		{"* * * 12 *", true},
		{"* * * 0 *", false},
		{"* * * 13 *", false},
		// Day of week: 0-6
		{"* * * * 0", true},
		{"* * * * 6", true},
		{"* * * * 7", false},
		// Ranges and steps
		{"0-30/5 * * * *", true},
		{"* 9-17 * * 1-5", true},
		{"*/2 */3 * * *", true},
	}
	for _, tt := range tests {
		ok, err := Validate(tt.expr)
		if tt.valid && (!ok || err != nil) {
			t.Errorf("Validate(%q) should be valid, got err: %v", tt.expr, err)
		}
		if !tt.valid && ok {
			t.Errorf("Validate(%q) should be invalid", tt.expr)
		}
	}
}

func TestValidate_Descriptors(t *testing.T) {
	descriptors := []string{"@hourly", "@daily", "@weekly", "@monthly", "@yearly", "@annually"}
	for _, d := range descriptors {
		ok, err := Validate(d)
		if !ok || err != nil {
			t.Errorf("Validate(%q) should be valid, got err: %v", d, err)
		}
	}
}

func TestNextRuns_Count(t *testing.T) {
	tests := []struct {
		expr string
		n    int
	}{
		{"* * * * *", 1},
		{"* * * * *", 5},
		{"* * * * *", 10},
	}
	for _, tt := range tests {
		runs, err := NextRuns(tt.expr, tt.n)
		if err != nil {
			t.Fatalf("NextRuns(%q, %d): %v", tt.expr, tt.n, err)
		}
		if len(runs) != tt.n {
			t.Errorf("NextRuns(%q, %d): got %d runs, want %d", tt.expr, tt.n, len(runs), tt.n)
		}
	}
}

func TestNextRuns_InvalidExpr(t *testing.T) {
	_, err := NextRuns("not-a-cron", 3)
	if err == nil {
		t.Error("NextRuns with invalid expression should return error")
	}
}

func TestNextRuns_Ascending(t *testing.T) {
	runs, err := NextRuns("* * * * *", 5)
	if err != nil {
		t.Fatalf("NextRuns: %v", err)
	}
	for i := 1; i < len(runs); i++ {
		if !runs[i].After(runs[i-1]) {
			t.Errorf("runs[%d] (%v) is not after runs[%d] (%v)", i, runs[i], i-1, runs[i-1])
		}
	}
}

func TestHumanReadable_PastTime(t *testing.T) {
	// A past time should get a formatted date string, not "in X ..."
	past := HumanReadable(fixedPastTime())
	if len(past) == 0 {
		t.Error("HumanReadable for past time should return non-empty string")
	}
}

func TestPreview_ValidResult(t *testing.T) {
	valid, runs, hr, errMsg := Preview("*/5 * * * *", 3)
	if !valid {
		t.Error("Preview should be valid")
	}
	if len(runs) != 3 {
		t.Errorf("expected 3 runs, got %d", len(runs))
	}
	if len(hr) != 3 {
		t.Errorf("expected 3 human-readable strings, got %d", len(hr))
	}
	if errMsg != "" {
		t.Errorf("error message should be empty, got %q", errMsg)
	}
}
