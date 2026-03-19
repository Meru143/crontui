package cli

import (
	"os/exec"
	"reflect"
	"testing"
)

type exitPanic struct {
	code int
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
	oldReadCrontab := cliReadCrontab
	oldExecCommand := cliExecCommand
	defer func() {
		exitCLI = oldExit
		cliReadCrontab = oldReadCrontab
		cliExecCommand = oldExecCommand
	}()

	exitCLI = func(code int) {
		panic(exitPanic{code: code})
	}
	cliReadCrontab = func() (string, error) {
		return "#* * * * * /bin/echo disabled\n", nil
	}
	executed := false
	cliExecCommand = func(command string) *exec.Cmd {
		executed = true
		return exec.Command("go", "version")
	}

	code := captureExitCode(t, func() {
		runNow([]string{"1"})
	})

	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if executed {
		t.Fatal("runNow should not execute disabled jobs")
	}
}
