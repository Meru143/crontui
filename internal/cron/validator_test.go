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
