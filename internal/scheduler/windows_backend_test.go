package scheduler

import (
	"strings"
	"testing"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

type mockPowerShellRunner struct {
	outputs []mockPowerShellResult
	scripts []string
}

type mockPowerShellResult struct {
	output string
	err    error
}

func (m *mockPowerShellRunner) Run(script string) ([]byte, error) {
	m.scripts = append(m.scripts, script)
	if len(m.outputs) == 0 {
		return nil, nil
	}

	result := m.outputs[0]
	m.outputs = m.outputs[1:]
	return []byte(result.output), result.err
}

func TestWindowsBackend_LoadJobs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WindowsTaskPath = `\CronTUI-Test\`

	exec1, args1 := wrapWindowsCommand(`Write-Output "first"`)
	exec2, args2 := wrapWindowsCommand(`Write-Output "second"`)

	runner := &mockPowerShellRunner{
		outputs: []mockPowerShellResult{
			{
				output: `[
  {
    "TaskName": "job-7",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("30 2 * * *") + `",
    "Description": "nightly",
    "Enabled": false,
    "Execute": "` + exec2 + `",
    "Arguments": "` + jsonEscape(args2) + `"
  },
  {
    "TaskName": "job-2",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("0 9 * * 1-5") + `",
    "Description": "weekday",
    "Enabled": true,
    "Execute": "` + exec1 + `",
    "Arguments": "` + jsonEscape(args1) + `"
  }
]`,
			},
		},
	}

	backend := &windowsBackend{cfg: cfg, runner: runner}

	jobs, err := backend.LoadJobs()
	if err != nil {
		t.Fatalf("LoadJobs returned error: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("job count = %d, want 2", len(jobs))
	}

	if jobs[0].ID != 2 || jobs[0].Schedule != "0 9 * * 1-5" || jobs[0].Command != `Write-Output "first"` || !jobs[0].Enabled {
		t.Fatalf("jobs[0] = %+v", jobs[0])
	}
	if jobs[1].ID != 7 || jobs[1].Schedule != "30 2 * * *" || jobs[1].Command != `Write-Output "second"` || jobs[1].Enabled {
		t.Fatalf("jobs[1] = %+v", jobs[1])
	}

	if len(runner.scripts) != 1 {
		t.Fatalf("script count = %d, want 1", len(runner.scripts))
	}
	if !strings.Contains(runner.scripts[0], `Get-ScheduledTask`) || !strings.Contains(runner.scripts[0], cfg.WindowsTaskPath) {
		t.Fatalf("list script = %q, want Get-ScheduledTask against %q", runner.scripts[0], cfg.WindowsTaskPath)
	}
}

func TestWindowsBackend_SaveJobsCreatesUpdatesAndDeletesTasks(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WindowsTaskPath = `\CronTUI-Test\`

	exec, args := wrapWindowsCommand(`Write-Output "existing"`)
	runner := &mockPowerShellRunner{
		outputs: []mockPowerShellResult{
			{
				output: `[
  {
    "TaskName": "job-1",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("0 9 * * 1-5") + `",
    "Description": "existing",
    "Enabled": true,
    "Execute": "` + exec + `",
    "Arguments": "` + jsonEscape(args) + `"
  },
  {
    "TaskName": "job-3",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("0 0 1 * *") + `",
    "Description": "remove me",
    "Enabled": true,
    "Execute": "` + exec + `",
    "Arguments": "` + jsonEscape(args) + `"
  }
]`,
			},
			{},
			{},
			{},
		},
	}

	backend := &windowsBackend{cfg: cfg, runner: runner}

	err := backend.SaveJobs(cfg, []types.CronJob{
		{ID: 1, Schedule: "30 2 * * *", Command: `Write-Output "updated"`, Description: "nightly", Enabled: true},
		{ID: 2, Schedule: "0 0 1 * *", Command: `Write-Output "new"`, Description: "monthly", Enabled: false},
	})
	if err != nil {
		t.Fatalf("SaveJobs returned error: %v", err)
	}

	if len(runner.scripts) != 4 {
		t.Fatalf("script count = %d, want 4", len(runner.scripts))
	}
	if !strings.Contains(runner.scripts[1], "Register-ScheduledTask") || !strings.Contains(runner.scripts[1], "job-1") {
		t.Fatalf("update script = %q, want Register-ScheduledTask for job-1", runner.scripts[1])
	}
	if !strings.Contains(runner.scripts[2], "Register-ScheduledTask") || !strings.Contains(runner.scripts[2], "job-2") {
		t.Fatalf("create script = %q, want Register-ScheduledTask for job-2", runner.scripts[2])
	}
	if !strings.Contains(runner.scripts[3], "Unregister-ScheduledTask") || !strings.Contains(runner.scripts[3], "job-3") {
		t.Fatalf("delete script = %q, want Unregister-ScheduledTask for job-3", runner.scripts[3])
	}
	if !strings.Contains(runner.scripts[2], encodeTaskSource("0 0 1 * *")) {
		t.Fatalf("create script = %q, want encoded source for monthly schedule", runner.scripts[2])
	}
}

func jsonEscape(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return replacer.Replace(value)
}
