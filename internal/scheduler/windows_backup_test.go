package scheduler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

func TestWindowsBackupCreateWritesManifestFromManagedJobs(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.BackupDir = t.TempDir()
	cfg.WindowsTaskPath = `\CronTUI-Test\`

	exec1, args1 := wrapWindowsCommand(`Write-Output "hello"`)
	exec2, args2 := wrapWindowsCommand(`Write-Output "nightly"`)

	runner := &mockPowerShellRunner{
		outputs: []mockPowerShellResult{
			{
				output: `[
  {
    "TaskName": "job-2",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("0 9 * * 1-5") + `",
    "Description": "weekday hello",
    "Enabled": true,
    "Execute": "` + exec1 + `",
    "Arguments": "` + jsonEscape(args1) + `"
  },
  {
    "TaskName": "job-7",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("30 2 * * *") + `",
    "Description": "nightly",
    "Enabled": false,
    "Execute": "` + exec2 + `",
    "Arguments": "` + jsonEscape(args2) + `"
  }
]`,
			},
		},
	}

	backend := &windowsBackend{cfg: cfg, runner: runner}

	path, err := backend.CreateBackup(cfg)
	if err != nil {
		t.Fatalf("CreateBackup returned error: %v", err)
	}
	if filepath.Dir(path) != cfg.BackupDir {
		t.Fatalf("backup dir = %q, want %q", filepath.Dir(path), cfg.BackupDir)
	}
	if !strings.HasPrefix(filepath.Base(path), "taskscheduler_") || !strings.HasSuffix(filepath.Base(path), ".bak") {
		t.Fatalf("backup filename = %q, want taskscheduler_*.bak", filepath.Base(path))
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}

	var manifest struct {
		Backend       string          `json:"backend"`
		SchemaVersion int             `json:"schema_version"`
		TaskPath      string          `json:"task_path"`
		Jobs          []types.CronJob `json:"jobs"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("backup manifest is not valid JSON: %v", err)
	}

	if manifest.Backend != "windows-task-scheduler" {
		t.Fatalf("manifest backend = %q, want windows-task-scheduler", manifest.Backend)
	}
	if manifest.SchemaVersion != 1 {
		t.Fatalf("manifest schema_version = %d, want 1", manifest.SchemaVersion)
	}
	if manifest.TaskPath != cfg.WindowsTaskPath {
		t.Fatalf("manifest task_path = %q, want %q", manifest.TaskPath, cfg.WindowsTaskPath)
	}
	if len(manifest.Jobs) != 2 {
		t.Fatalf("manifest job count = %d, want 2", len(manifest.Jobs))
	}
	if manifest.Jobs[0].ID != 2 || manifest.Jobs[1].ID != 7 {
		t.Fatalf("manifest jobs = %#v, want IDs [2 7]", manifest.Jobs)
	}
}

func TestWindowsListBackupsReturnsManifestMetadata(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.BackupDir = t.TempDir()

	writeManifest := func(name string, jobs []types.CronJob) {
		t.Helper()
		data, err := json.Marshal(windowsBackupManifest{
			Backend:       windowsBackupBackend,
			SchemaVersion: windowsBackupSchemaVersion,
			TaskPath:      `\CronTUI-Test\`,
			Jobs:          jobs,
		})
		if err != nil {
			t.Fatalf("Marshal manifest: %v", err)
		}
		path := filepath.Join(cfg.BackupDir, name)
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("WriteFile(%q): %v", path, err)
		}
	}

	writeManifest("taskscheduler_20260319_100000.bak", []types.CronJob{{ID: 1}, {ID: 2}})
	writeManifest("taskscheduler_20260318_100000.bak", []types.CronJob{{ID: 9}})
	if err := os.WriteFile(filepath.Join(cfg.BackupDir, "crontab_20260319_100000.bak"), []byte("0 * * * * cmd\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(non-windows backup): %v", err)
	}

	newer := filepath.Join(cfg.BackupDir, "taskscheduler_20260319_100000.bak")
	older := filepath.Join(cfg.BackupDir, "taskscheduler_20260318_100000.bak")
	if err := os.Chtimes(older, time.Date(2026, time.March, 18, 10, 0, 0, 0, time.UTC), time.Date(2026, time.March, 18, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Chtimes older: %v", err)
	}
	if err := os.Chtimes(newer, time.Date(2026, time.March, 19, 10, 0, 0, 0, time.UTC), time.Date(2026, time.March, 19, 10, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Chtimes newer: %v", err)
	}

	backend := &windowsBackend{cfg: cfg}
	backups, err := backend.ListBackups(cfg)
	if err != nil {
		t.Fatalf("ListBackups returned error: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("backup count = %d, want 2", len(backups))
	}
	if backups[0].Filename != "taskscheduler_20260319_100000.bak" || backups[0].JobCount != 2 {
		t.Fatalf("backups[0] = %+v", backups[0])
	}
	if backups[1].Filename != "taskscheduler_20260318_100000.bak" || backups[1].JobCount != 1 {
		t.Fatalf("backups[1] = %+v", backups[1])
	}
}

func TestWindowsRestoreBackupCreatesPreRestoreBackupAndReplacesManagedTaskSet(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.BackupDir = t.TempDir()
	cfg.WindowsTaskPath = `\CronTUI-Test\`

	restoreData, err := json.Marshal(windowsBackupManifest{
		Backend:       windowsBackupBackend,
		SchemaVersion: windowsBackupSchemaVersion,
		TaskPath:      cfg.WindowsTaskPath,
		Jobs: []types.CronJob{
			{ID: 1, Schedule: "30 2 * * *", Command: `Write-Output "restored-1"`, Description: "nightly", Enabled: true},
			{ID: 2, Schedule: "0 0 1 * *", Command: `Write-Output "restored-2"`, Description: "monthly", Enabled: false},
		},
	})
	if err != nil {
		t.Fatalf("Marshal restore manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cfg.BackupDir, "restore-source.bak"), restoreData, 0o644); err != nil {
		t.Fatalf("WriteFile(restore-source): %v", err)
	}

	currentExec, currentArgs := wrapWindowsCommand(`Write-Output "current"`)
	currentJSON := `[
  {
    "TaskName": "job-9",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("0 9 * * 1-5") + `",
    "Description": "current",
    "Enabled": true,
    "Execute": "` + currentExec + `",
    "Arguments": "` + jsonEscape(currentArgs) + `"
  }
]`

	runner := &mockPowerShellRunner{
		outputs: []mockPowerShellResult{
			{output: currentJSON},
			{output: currentJSON},
			{},
			{},
			{},
		},
	}
	backend := &windowsBackend{cfg: cfg, runner: runner}

	if err := backend.RestoreBackup(cfg, "restore-source.bak"); err != nil {
		t.Fatalf("RestoreBackup returned error: %v", err)
	}

	backupFiles, err := filepath.Glob(filepath.Join(cfg.BackupDir, "taskscheduler_*.bak"))
	if err != nil {
		t.Fatalf("Glob pre-restore backups: %v", err)
	}
	if len(backupFiles) != 1 {
		t.Fatalf("pre-restore backup count = %d, want 1", len(backupFiles))
	}

	preRestoreData, err := os.ReadFile(backupFiles[0])
	if err != nil {
		t.Fatalf("ReadFile(pre-restore): %v", err)
	}
	var preRestore windowsBackupManifest
	if err := json.Unmarshal(preRestoreData, &preRestore); err != nil {
		t.Fatalf("pre-restore backup JSON: %v", err)
	}
	if len(preRestore.Jobs) != 1 || preRestore.Jobs[0].ID != 9 {
		t.Fatalf("pre-restore jobs = %#v, want current job #9", preRestore.Jobs)
	}

	if len(runner.scripts) != 5 {
		t.Fatalf("script count = %d, want 5", len(runner.scripts))
	}
	if !strings.Contains(runner.scripts[2], "Register-ScheduledTask") || !strings.Contains(runner.scripts[2], "job-1") {
		t.Fatalf("restore create script = %q, want Register-ScheduledTask for job-1", runner.scripts[2])
	}
	if !strings.Contains(runner.scripts[3], "Register-ScheduledTask") || !strings.Contains(runner.scripts[3], "job-2") {
		t.Fatalf("restore create script = %q, want Register-ScheduledTask for job-2", runner.scripts[3])
	}
	if !strings.Contains(runner.scripts[4], "Unregister-ScheduledTask") || !strings.Contains(runner.scripts[4], "job-9") {
		t.Fatalf("restore delete script = %q, want Unregister-ScheduledTask for job-9", runner.scripts[4])
	}
}

func TestWindowsRemoveAllCreatesBackupAndDeletesOnlyConfiguredPathTasks(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.BackupDir = t.TempDir()
	cfg.WindowsTaskPath = `\CronTUI-Test\`

	exec1, args1 := wrapWindowsCommand(`Write-Output "one"`)
	exec2, args2 := wrapWindowsCommand(`Write-Output "two"`)
	currentJSON := `[
  {
    "TaskName": "job-1",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("0 9 * * 1-5") + `",
    "Description": "one",
    "Enabled": true,
    "Execute": "` + exec1 + `",
    "Arguments": "` + jsonEscape(args1) + `"
  },
  {
    "TaskName": "job-2",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("30 2 * * *") + `",
    "Description": "two",
    "Enabled": false,
    "Execute": "` + exec2 + `",
    "Arguments": "` + jsonEscape(args2) + `"
  }
]`

	runner := &mockPowerShellRunner{
		outputs: []mockPowerShellResult{
			{output: currentJSON},
			{output: currentJSON},
			{},
			{},
		},
	}
	backend := &windowsBackend{cfg: cfg, runner: runner}

	if err := backend.RemoveAll(cfg); err != nil {
		t.Fatalf("RemoveAll returned error: %v", err)
	}

	backupFiles, err := filepath.Glob(filepath.Join(cfg.BackupDir, "taskscheduler_*.bak"))
	if err != nil {
		t.Fatalf("Glob backups: %v", err)
	}
	if len(backupFiles) != 1 {
		t.Fatalf("backup count = %d, want 1", len(backupFiles))
	}

	if len(runner.scripts) != 4 {
		t.Fatalf("script count = %d, want 4", len(runner.scripts))
	}
	for i, script := range runner.scripts {
		if !strings.Contains(script, cfg.WindowsTaskPath) {
			t.Fatalf("script[%d] = %q, want only configured task path %q", i, script, cfg.WindowsTaskPath)
		}
	}
	if !strings.Contains(runner.scripts[2], "Unregister-ScheduledTask") || !strings.Contains(runner.scripts[2], "job-1") {
		t.Fatalf("remove-all script = %q, want Unregister-ScheduledTask for job-1", runner.scripts[2])
	}
	if !strings.Contains(runner.scripts[3], "Unregister-ScheduledTask") || !strings.Contains(runner.scripts[3], "job-2") {
		t.Fatalf("remove-all script = %q, want Unregister-ScheduledTask for job-2", runner.scripts[3])
	}
}

func TestWindowsRunNowExecutesSavedCommandDirectly(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.WindowsTaskPath = `\CronTUI-Test\`

	execCmd, args := wrapWindowsCommand(`Write-Output "saved"`)
	runner := &mockPowerShellRunner{
		outputs: []mockPowerShellResult{
			{
				output: `[
  {
    "TaskName": "job-4",
    "TaskPath": "\\CronTUI-Test\\",
    "Source": "` + encodeTaskSource("0 9 * * 1-5") + `",
    "Description": "saved command",
    "Enabled": true,
    "Execute": "` + execCmd + `",
    "Arguments": "` + jsonEscape(args) + `"
  }
]`,
			},
		},
	}

	oldRunImmediateCommand := runImmediateCommand
	defer func() { runImmediateCommand = oldRunImmediateCommand }()

	var gotOS, gotCommand string
	runImmediateCommand = func(goos, command string) ([]byte, error) {
		gotOS = goos
		gotCommand = command
		return []byte("direct output"), nil
	}

	backend := &windowsBackend{cfg: cfg, runner: runner}
	out, err := backend.RunNow(4)
	if err != nil {
		t.Fatalf("RunNow returned error: %v", err)
	}
	if string(out) != "direct output" {
		t.Fatalf("RunNow output = %q, want %q", out, "direct output")
	}
	if gotOS != "windows" {
		t.Fatalf("runImmediateCommand goos = %q, want windows", gotOS)
	}
	if gotCommand != `Write-Output "saved"` {
		t.Fatalf("runImmediateCommand command = %q, want saved command", gotCommand)
	}
}
