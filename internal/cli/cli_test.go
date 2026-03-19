package cli

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

type exitPanic struct {
	code int
}

type stubBackend struct {
	loadJobsFn                func() ([]types.CronJob, error)
	saveJobsFn                func(config.Config, []types.CronJob) error
	createBackupFn            func(config.Config) (string, error)
	listBackupsFn             func(config.Config) ([]types.Backup, error)
	restoreBackupFn           func(config.Config, string) error
	removeAllFn               func(config.Config) error
	runNowFn                  func(int) ([]byte, error)
	validateManagedScheduleFn func(string) error
}

func (s stubBackend) LoadJobs() ([]types.CronJob, error) {
	if s.loadJobsFn != nil {
		return s.loadJobsFn()
	}
	return nil, nil
}

func (s stubBackend) SaveJobs(cfg config.Config, jobs []types.CronJob) error {
	if s.saveJobsFn != nil {
		return s.saveJobsFn(cfg, jobs)
	}
	return nil
}

func (s stubBackend) CreateBackup(cfg config.Config) (string, error) {
	if s.createBackupFn != nil {
		return s.createBackupFn(cfg)
	}
	return "", nil
}

func (s stubBackend) ListBackups(cfg config.Config) ([]types.Backup, error) {
	if s.listBackupsFn != nil {
		return s.listBackupsFn(cfg)
	}
	return nil, nil
}

func (s stubBackend) RestoreBackup(cfg config.Config, filename string) error {
	if s.restoreBackupFn != nil {
		return s.restoreBackupFn(cfg, filename)
	}
	return nil
}

func (s stubBackend) RemoveAll(cfg config.Config) error {
	if s.removeAllFn != nil {
		return s.removeAllFn(cfg)
	}
	return nil
}

func (s stubBackend) RunNow(id int) ([]byte, error) {
	if s.runNowFn != nil {
		return s.runNowFn(id)
	}
	return nil, nil
}

func (s stubBackend) ValidateManagedSchedule(expr string) error {
	if s.validateManagedScheduleFn != nil {
		return s.validateManagedScheduleFn(expr)
	}
	return nil
}

func captureExitCode(t *testing.T, fn func()) (code int) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			exit, ok := r.(exitPanic)
			if !ok {
				panic(r)
			}
			code = exit.code
			return
		}
		t.Fatal("expected function to exit")
	}()

	fn()
	return 0
}

func TestParseInvocation_DebugBeforeCommand(t *testing.T) {
	cmd, subArgs, debug, ok := parseInvocation([]string{"crontui", "--debug", "completion", "bash"})
	if !ok {
		t.Fatal("parseInvocation should handle a leading --debug flag")
	}
	if !debug {
		t.Fatal("parseInvocation should report debug=true when --debug is present")
	}
	if cmd != "completion" {
		t.Fatalf("cmd = %q, want %q", cmd, "completion")
	}
	if !reflect.DeepEqual(subArgs, []string{"bash"}) {
		t.Fatalf("subArgs = %#v, want %#v", subArgs, []string{"bash"})
	}
}

func TestParsePreviewArgs_RejectsNonPositiveCount(t *testing.T) {
	for _, args := range [][]string{
		{"0 * * * *", "0"},
		{"0 * * * *", "-1"},
	} {
		if _, _, err := parsePreviewArgs(args); err == nil {
			t.Fatalf("parsePreviewArgs(%#v) should reject non-positive counts", args)
		}
	}
}

func TestRun_UnknownCommandExits(t *testing.T) {
	oldExit := exitCLI
	defer func() { exitCLI = oldExit }()

	exitCLI = func(code int) {
		panic(exitPanic{code: code})
	}

	code := captureExitCode(t, func() {
		Run([]string{"crontui", "does-not-exist"})
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestRunNow_RejectsDisabledJobs(t *testing.T) {
	oldExit := exitCLI
	oldBackend := cliBackendFn
	defer func() {
		exitCLI = oldExit
		cliBackendFn = oldBackend
	}()

	exitCLI = func(code int) {
		panic(exitPanic{code: code})
	}
	cliBackendFn = func(config.Config) backend {
		return stubBackend{
			runNowFn: func(id int) ([]byte, error) {
				return nil, errors.New("job #1 is disabled")
			},
		}
	}

	code := captureExitCode(t, func() {
		runNow(config.DefaultConfig(), []string{"1"})
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}

func TestWindowsRunListUsesSchedulerBackend(t *testing.T) {
	oldBackend := cliBackendFn
	defer func() { cliBackendFn = oldBackend }()

	loaded := false
	cliBackendFn = func(config.Config) backend {
		return stubBackend{
			loadJobsFn: func() ([]types.CronJob, error) {
				loaded = true
				return []types.CronJob{
					{ID: 7, Schedule: "0 9 * * 1-5", Command: `Write-Output "hello"`, Description: "weekday", Enabled: true},
				}, nil
			},
		}
	}

	output := captureStdout(t, func() {
		runList(config.DefaultConfig(), []string{"--json"})
	})

	if !loaded {
		t.Fatal("runList should load jobs from the scheduler backend")
	}
	if !bytes.Contains([]byte(output), []byte(`"ID": 7`)) {
		t.Fatalf("runList output = %q, want JSON with backend job", output)
	}
}

func TestWindowsRunAddRejectsUnsupportedBackendSchedule(t *testing.T) {
	oldExit := exitCLI
	oldBackend := cliBackendFn
	defer func() {
		exitCLI = oldExit
		cliBackendFn = oldBackend
	}()

	exitCLI = func(code int) {
		panic(exitPanic{code: code})
	}

	saved := false
	cliBackendFn = func(config.Config) backend {
		return stubBackend{
			loadJobsFn: func() ([]types.CronJob, error) { return nil, nil },
			validateManagedScheduleFn: func(expr string) error {
				return errors.New("windows backend does not support @reboot without elevated Task Scheduler permissions")
			},
			saveJobsFn: func(config.Config, []types.CronJob) error {
				saved = true
				return nil
			},
		}
	}

	code := captureExitCode(t, func() {
		runAdd(config.DefaultConfig(), []string{"@reboot", `Write-Output "hello"`})
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if saved {
		t.Fatal("runAdd should not save when the backend rejects the schedule")
	}
}

func TestWindowsRunNowUsesSchedulerBackend(t *testing.T) {
	oldBackend := cliBackendFn
	defer func() { cliBackendFn = oldBackend }()

	var ranID int
	cliBackendFn = func(config.Config) backend {
		return stubBackend{
			runNowFn: func(id int) ([]byte, error) {
				ranID = id
				return []byte("backend output\n"), nil
			},
		}
	}

	output := captureStdout(t, func() {
		runNow(config.DefaultConfig(), []string{"42"})
	})

	if ranID != 42 {
		t.Fatalf("RunNow ID = %d, want 42", ranID)
	}
	if !bytes.Contains([]byte(output), []byte("backend output")) {
		t.Fatalf("runNow output = %q, want backend output", output)
	}
}

func TestWindowsRunBackupAndRestoreUseSchedulerBackend(t *testing.T) {
	oldBackend := cliBackendFn
	defer func() { cliBackendFn = oldBackend }()

	var restored string
	cliBackendFn = func(config.Config) backend {
		return stubBackend{
			createBackupFn: func(config.Config) (string, error) {
				return `C:\tmp\taskscheduler_20260319_120000.bak`, nil
			},
			restoreBackupFn: func(cfg config.Config, filename string) error {
				restored = filename
				return nil
			},
		}
	}

	backupOutput := captureStdout(t, func() {
		runBackup(config.DefaultConfig())
	})
	restoreOutput := captureStdout(t, func() {
		runRestore(config.DefaultConfig(), []string{"taskscheduler_20260319_120000.bak"})
	})

	if restored != "taskscheduler_20260319_120000.bak" {
		t.Fatalf("restored filename = %q, want backup filename", restored)
	}
	if !bytes.Contains([]byte(backupOutput), []byte("Backup created")) {
		t.Fatalf("backup output = %q, want success text", backupOutput)
	}
	if !bytes.Contains([]byte(restoreOutput), []byte("Restored from taskscheduler_20260319_120000.bak")) {
		t.Fatalf("restore output = %q, want success text", restoreOutput)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := stdout
	defer func() { stdout = oldStdout }()

	var buf bytes.Buffer
	stdout = &buf
	fn()
	return buf.String()
}
