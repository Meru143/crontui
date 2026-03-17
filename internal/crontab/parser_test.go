package crontab

import "testing"

func TestParseCrontab(t *testing.T) {
	raw := `# description: Hourly backup
0 * * * * /usr/bin/backup

# description: Disabled cleanup
#0 0 * * * /usr/bin/cleanup

@reboot /usr/bin/startup
SHELL=/bin/bash
`

	jobs := ParseCrontab(raw)

	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}

	// First job: enabled hourly
	if jobs[0].Schedule != "0 * * * *" {
		t.Errorf("job[0].Schedule = %q, want %q", jobs[0].Schedule, "0 * * * *")
	}
	if jobs[0].Command != "/usr/bin/backup" {
		t.Errorf("job[0].Command = %q, want %q", jobs[0].Command, "/usr/bin/backup")
	}
	if !jobs[0].Enabled {
		t.Error("job[0] should be enabled")
	}
	if jobs[0].Description != "Hourly backup" {
		t.Errorf("job[0].Description = %q, want %q", jobs[0].Description, "Hourly backup")
	}

	// Second job: disabled
	if jobs[1].Enabled {
		t.Error("job[1] should be disabled")
	}
	if jobs[1].Description != "Disabled cleanup" {
		t.Errorf("job[1].Description = %q, want %q", jobs[1].Description, "Disabled cleanup")
	}

	// Third job: @reboot
	if jobs[2].Schedule != "@reboot" {
		t.Errorf("job[2].Schedule = %q, want %q", jobs[2].Schedule, "@reboot")
	}
	if jobs[2].Command != "/usr/bin/startup" {
		t.Errorf("job[2].Command = %q, want %q", jobs[2].Command, "/usr/bin/startup")
	}
}

func TestIsCronLine(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"* * * * * cmd", true},
		{"0 * * * * cmd", true},
		{"@reboot cmd", false},
		{"# comment", false},
		{"SHELL=/bin/bash", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isCronLine(tt.line)
		if got != tt.want {
			t.Errorf("isCronLine(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestIsEnvVar(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{"SHELL=/bin/bash", true},
		{"PATH=/usr/bin", true},
		{"_var=value", true},
		{"0 * * * * cmd", false},
		{"# comment", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isEnvVar(tt.line)
		if got != tt.want {
			t.Errorf("isEnvVar(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}
