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

func TestParseCrontab_Annotations(t *testing.T) {
	raw := `# description: My job
# workingdir: /home/user
# mailto: admin@example.com
*/10 * * * * /usr/bin/task
`
	jobs := ParseCrontab(raw)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	j := jobs[0]
	if j.Description != "My job" {
		t.Errorf("Description = %q, want %q", j.Description, "My job")
	}
	if j.WorkingDir != "/home/user" {
		t.Errorf("WorkingDir = %q, want %q", j.WorkingDir, "/home/user")
	}
	if j.Mailto != "admin@example.com" {
		t.Errorf("Mailto = %q, want %q", j.Mailto, "admin@example.com")
	}
}

func TestParseCrontab_BlankLineResetsAnnotations(t *testing.T) {
	raw := `# description: Should be discarded
# workingdir: /tmp

0 * * * * /usr/bin/noop
`
	jobs := ParseCrontab(raw)
	if len(jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(jobs))
	}
	if jobs[0].Description != "" {
		t.Errorf("Description should be empty after blank line, got %q", jobs[0].Description)
	}
	if jobs[0].WorkingDir != "" {
		t.Errorf("WorkingDir should be empty after blank line, got %q", jobs[0].WorkingDir)
	}
}

func TestParseCrontab_SequentialIDs(t *testing.T) {
	raw := `0 * * * * /usr/bin/a
0 * * * * /usr/bin/b
0 * * * * /usr/bin/c
`
	jobs := ParseCrontab(raw)
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
	for i, j := range jobs {
		want := i + 1
		if j.ID != want {
			t.Errorf("job[%d].ID = %d, want %d", i, j.ID, want)
		}
	}
}

func TestParseCrontab_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{"empty string", "", 0},
		{"only whitespace", "   \n\t\n  ", 0},
		{"only comments", "# first comment\n# second comment\n", 0},
		{"only env vars", "SHELL=/bin/bash\nPATH=/usr/bin\n", 0},
		{"extra spaces", "  0 * * * *   /usr/bin/cmd  \n", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobs := ParseCrontab(tt.raw)
			if len(jobs) != tt.want {
				t.Errorf("got %d jobs, want %d", len(jobs), tt.want)
			}
		})
	}
}

func TestParseCrontab_SpecialCharacters(t *testing.T) {
	raw := `0 * * * * /usr/bin/cmd --flag | tee /var/log/out.log
0 0 * * * echo "hello world" > /tmp/hello.txt 2>&1
0 0 * * * /bin/a; /bin/b; /bin/c
`
	jobs := ParseCrontab(raw)
	if len(jobs) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(jobs))
	}
	if jobs[0].Command != "/usr/bin/cmd --flag | tee /var/log/out.log" {
		t.Errorf("pipe command: got %q", jobs[0].Command)
	}
	if jobs[1].Command != `echo "hello world" > /tmp/hello.txt 2>&1` {
		t.Errorf("redirect command: got %q", jobs[1].Command)
	}
	if jobs[2].Command != "/bin/a; /bin/b; /bin/c" {
		t.Errorf("semicolon command: got %q", jobs[2].Command)
	}
}
