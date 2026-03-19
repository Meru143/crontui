package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/meru143/crontui/internal/completion"
	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/internal/cron"
	"github.com/meru143/crontui/internal/crontab"
	"github.com/meru143/crontui/internal/scheduler"
	"github.com/meru143/crontui/internal/version"
	"github.com/meru143/crontui/pkg/types"
)

// Debug controls verbose logging output.
var Debug bool

type backend = scheduler.Backend

var (
	cliBackendFn           = scheduler.DefaultBackend
	stdout       io.Writer = os.Stdout
	stderr       io.Writer = os.Stderr
	exitCLI                = os.Exit
)

func handleWriteError(err error) {
	if err == nil {
		return
	}

	_, _ = fmt.Fprintf(stderr, "I/O error: %v\n", err)
	exitCLI(1)
}

func write(w io.Writer, text string) {
	_, err := io.WriteString(w, text)
	handleWriteError(err)
}

func writef(w io.Writer, format string, args ...any) {
	_, err := fmt.Fprintf(w, format, args...)
	handleWriteError(err)
}

func writeln(w io.Writer, args ...any) {
	_, err := fmt.Fprintln(w, args...)
	handleWriteError(err)
}

func flushWriter(w *tabwriter.Writer) {
	handleWriteError(w.Flush())
}

func parseInvocation(args []string) (cmd string, subArgs []string, debug bool, ok bool) {
	if len(args) < 2 {
		return "", nil, false, false
	}

	filtered := make([]string, 0, len(args)-1)
	for _, a := range args[1:] {
		if a == "--debug" {
			debug = true
			continue
		}
		filtered = append(filtered, a)
	}

	if len(filtered) == 0 {
		return "", nil, debug, false
	}

	return filtered[0], filtered[1:], debug, true
}

func parsePreviewArgs(args []string) (expr string, count int, err error) {
	if len(args) < 1 {
		return "", 0, fmt.Errorf("usage: crontui preview <expression> [count]")
	}

	count = 10
	expr = args[0]

	if len(args) == 1 {
		return expr, count, nil
	}

	if n, parseErr := strconv.Atoi(args[len(args)-1]); parseErr == nil {
		if n <= 0 {
			return "", 0, fmt.Errorf("count must be greater than 0")
		}
		return strings.Join(args[:len(args)-1], " "), n, nil
	}

	return strings.Join(args, " "), count, nil
}

// Run processes CLI subcommands. Returns true if a subcommand was handled.
func Run(args []string) bool {
	cmd, subArgs, debug, ok := parseInvocation(args)
	if !ok {
		return false
	}

	Debug = debug
	if Debug {
		log.SetOutput(stderr)
		log.SetPrefix("[debug] ")
		log.Println("debug mode enabled")
	}

	cfg, err := config.Load()
	if err != nil {
		writef(stderr, "Error loading config: %v\n", err)
		exitCLI(1)
		return true
	}

	switch cmd {
	case "list", "ls":
		return runList(cfg, subArgs)
	case "add":
		return runAdd(cfg, subArgs)
	case "delete", "rm":
		return runDelete(cfg, subArgs)
	case "enable":
		return runToggle(cfg, subArgs, true)
	case "disable":
		return runToggle(cfg, subArgs, false)
	case "validate":
		return runValidate(subArgs)
	case "preview":
		return runPreview(subArgs)
	case "backup":
		return runBackup(cfg)
	case "restore":
		return runRestore(cfg, subArgs)
	case "export":
		return runExport(cfg, subArgs)
	case "import":
		return runImport(cfg, subArgs)
	case "runnow", "run":
		return runNow(cfg, subArgs)
	case "completion", "--completion":
		return runCompletion(subArgs)
	case "help", "--help", "-h":
		printHelp()
		return true
	case "version", "--version", "-v":
		writeln(stdout, version.Full())
		return true
	default:
		writef(stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		exitCLI(1)
		return true
	}
}

func runList(cfg config.Config, args []string) bool {
	jobs, err := cliBackendFn(cfg).LoadJobs()
	if err != nil {
		writef(stderr, "Error: %v\n", err)
		exitCLI(1)
	}

	// Check for --json flag
	for _, a := range args {
		if a == "--json" {
			data, _ := json.MarshalIndent(jobs, "", "  ")
			writeln(stdout, string(data))
			return true
		}
	}

	w := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	writeln(w, "ID\tStatus\tSchedule\tCommand\tDescription")
	writeln(w, "--\t------\t--------\t-------\t-----------")

	for _, job := range jobs {
		status := "ON "
		if !job.Enabled {
			status = "OFF"
		}
		writef(w, "%d\t%s\t%s\t%s\t%s\n", job.ID, status, job.Schedule, job.Command, job.Description)
	}
	flushWriter(w)
	return true
}

func runAdd(cfg config.Config, args []string) bool {
	if len(args) < 2 {
		writef(stderr, "Usage: crontui add <schedule> <command> [--desc \"description\"]\n")
		exitCLI(1)
	}

	schedule := args[0]
	command := args[1]
	description := ""
	backend := cliBackendFn(cfg)

	for i, a := range args {
		if a == "--desc" && i+1 < len(args) {
			description = args[i+1]
			break
		}
	}

	// Validate schedule
	valid, err := cron.Validate(schedule)
	if !valid {
		writef(stderr, "Invalid schedule: %v\n", err)
		exitCLI(1)
	}
	if err := backend.ValidateManagedSchedule(schedule); err != nil {
		writef(stderr, "Unsupported schedule on this platform: %v\n", err)
		exitCLI(1)
	}

	jobs, err := backend.LoadJobs()
	if err != nil {
		writef(stderr, "Error loading jobs: %v\n", err)
		exitCLI(1)
	}

	maxID := 0
	for _, j := range jobs {
		if j.ID > maxID {
			maxID = j.ID
		}
	}

	jobs = append(jobs, types.CronJob{
		ID:          maxID + 1,
		Schedule:    schedule,
		Command:     command,
		Description: description,
		Enabled:     true,
	})

	if err := backend.SaveJobs(cfg, jobs); err != nil {
		writef(stderr, "Error saving jobs: %v\n", err)
		exitCLI(1)
	}

	writef(stdout, "Added job #%d: %s %s\n", maxID+1, schedule, command)
	return true
}

func runDelete(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		writef(stderr, "Usage: crontui delete <job-id>\n")
		exitCLI(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		writef(stderr, "Invalid job ID: %s\n", args[0])
		exitCLI(1)
	}

	backend := cliBackendFn(cfg)
	jobs, err := backend.LoadJobs()
	if err != nil {
		writef(stderr, "Error: %v\n", err)
		exitCLI(1)
	}
	found := false
	newJobs := make([]types.CronJob, 0)
	for _, j := range jobs {
		if j.ID == id {
			found = true
			continue
		}
		newJobs = append(newJobs, j)
	}

	if !found {
		writef(stderr, "Job #%d not found\n", id)
		exitCLI(1)
	}

	if err := backend.SaveJobs(cfg, newJobs); err != nil {
		writef(stderr, "Error: %v\n", err)
		exitCLI(1)
	}

	writef(stdout, "Deleted job #%d\n", id)
	return true
}

func runToggle(cfg config.Config, args []string, enable bool) bool {
	if len(args) < 1 {
		action := "enable"
		if !enable {
			action = "disable"
		}
		writef(stderr, "Usage: crontui %s <job-id>\n", action)
		exitCLI(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		writef(stderr, "Invalid job ID: %s\n", args[0])
		exitCLI(1)
	}

	backend := cliBackendFn(cfg)
	jobs, err := backend.LoadJobs()
	if err != nil {
		writef(stderr, "Error: %v\n", err)
		exitCLI(1)
	}
	found := false
	for i := range jobs {
		if jobs[i].ID == id {
			jobs[i].Enabled = enable
			found = true
			break
		}
	}

	if !found {
		writef(stderr, "Job #%d not found\n", id)
		exitCLI(1)
	}

	if err := backend.SaveJobs(cfg, jobs); err != nil {
		writef(stderr, "Error: %v\n", err)
		exitCLI(1)
	}

	action := "enabled"
	if !enable {
		action = "disabled"
	}
	writef(stdout, "Job #%d %s\n", id, action)
	return true
}

func runValidate(args []string) bool {
	if len(args) < 1 {
		writef(stderr, "Usage: crontui validate <expression>\n")
		exitCLI(1)
	}

	expr := strings.Join(args, " ")
	valid, err := cron.Validate(expr)
	if valid {
		writef(stdout, "✓ Valid: %s\n", expr)

		// Show next runs
		if notice, ok := cron.PreviewNotice(expr); ok {
			writef(stdout, "\nNext runs:\n  %s\n", notice)
			return true
		}

		runs, _ := cron.NextRuns(expr, 5)
		if len(runs) > 0 {
			writeln(stdout, "\nNext runs:")
			for i, t := range runs {
				writef(stdout, "  %d. %s  (%s)\n", i+1, t.Format("Mon Jan 2 15:04:05 2006"), cron.HumanReadable(t))
			}
		}
	} else {
		writef(stderr, "✗ Invalid: %v\n", err)
		exitCLI(1)
	}
	return true
}

func runPreview(args []string) bool {
	expr, count, err := parsePreviewArgs(args)
	if err != nil {
		if len(args) < 1 {
			writef(stderr, "Usage: crontui preview <expression> [count]\n")
		} else {
			writef(stderr, "Invalid preview arguments: %v\n", err)
		}
		exitCLI(1)
	}

	if notice, ok := cron.PreviewNotice(expr); ok {
		writef(stdout, "Next %d runs for: %s\n\n", count, expr)
		writef(stdout, "  %s\n", notice)
		return true
	}

	runs, err := cron.NextRuns(expr, count)
	if err != nil {
		writef(stderr, "Invalid expression: %v\n", err)
		exitCLI(1)
	}

	writef(stdout, "Next %d runs for: %s\n\n", count, expr)
	for i, t := range runs {
		writef(stdout, "  %d. %s  (%s)\n", i+1, t.Format("Mon Jan 2 15:04:05 2006"), cron.HumanReadable(t))
	}
	return true
}

func runBackup(cfg config.Config) bool {
	path, err := cliBackendFn(cfg).CreateBackup(cfg)
	if err != nil {
		writef(stderr, "Backup failed: %v\n", err)
		exitCLI(1)
	}
	writef(stdout, "Backup created: %s\n", path)
	return true
}

func runRestore(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		writef(stderr, "Usage: crontui restore <filename>\n")
		exitCLI(1)
	}
	if err := cliBackendFn(cfg).RestoreBackup(cfg, args[0]); err != nil {
		writef(stderr, "Restore failed: %v\n", err)
		exitCLI(1)
	}
	writef(stdout, "Restored from %s\n", args[0])
	return true
}

func runExport(cfg config.Config, args []string) bool {
	jobs, err := cliBackendFn(cfg).LoadJobs()
	if err != nil {
		writef(stderr, "Error: %v\n", err)
		exitCLI(1)
	}

	// Check for --format=json
	format := "json"
	for _, a := range args {
		if strings.HasPrefix(a, "--format=") {
			format = strings.TrimPrefix(a, "--format=")
		}
	}

	switch format {
	case "json":
		data, _ := json.MarshalIndent(jobs, "", "  ")
		writeln(stdout, string(data))
	case "crontab":
		write(stdout, crontab.FormatCrontab(jobs))
	default:
		writef(stderr, "Unknown format: %s\n", format)
		exitCLI(1)
	}
	return true
}

func runImport(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		writef(stderr, "Usage: crontui import <file.json>\n")
		exitCLI(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		writef(stderr, "Error reading file: %v\n", err)
		exitCLI(1)
	}

	var jobs []types.CronJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		writef(stderr, "Error parsing JSON: %v\n", err)
		exitCLI(1)
	}

	backend := cliBackendFn(cfg)
	for _, job := range jobs {
		valid, err := cron.Validate(job.Schedule)
		if !valid || err != nil {
			writef(stderr, "Invalid schedule for job #%d: %v\n", job.ID, err)
			exitCLI(1)
		}
		if err := backend.ValidateManagedSchedule(job.Schedule); err != nil {
			writef(stderr, "Unsupported schedule for job #%d on this platform: %v\n", job.ID, err)
			exitCLI(1)
		}
	}

	if err := backend.SaveJobs(cfg, jobs); err != nil {
		writef(stderr, "Error writing jobs: %v\n", err)
		exitCLI(1)
	}

	writef(stdout, "Imported %d jobs\n", len(jobs))
	return true
}

func runNow(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		writef(stderr, "Usage: crontui runnow <job-id>\n")
		exitCLI(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		writef(stderr, "Invalid job ID: %s\n", args[0])
		exitCLI(1)
	}

	writef(stdout, "Running job #%d\n\n", id)

	out, err := cliBackendFn(cfg).RunNow(id)
	if len(out) > 0 {
		write(stdout, string(out))
		if !strings.HasSuffix(string(out), "\n") {
			writeln(stdout)
		}
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			writef(stdout, "\nJob exited with code %d\n", exitErr.ExitCode())
		} else {
			writef(stderr, "\nExecution error: %v\n", err)
		}
		exitCLI(1)
	}

	if len(out) == 0 {
		writeln(stdout, "(no output)")
	}
	writeln(stdout, "\nJob completed successfully")
	return true
}

func runCompletion(args []string) bool {
	shell := "bash"
	if len(args) > 0 {
		shell = args[0]
	}

	switch shell {
	case "bash":
		write(stdout, completion.Bash())
	case "zsh":
		write(stdout, completion.Zsh())
	case "fish":
		write(stdout, completion.Fish())
	default:
		writef(stderr, "Unsupported shell: %s (supported: bash, zsh, fish)\n", shell)
		exitCLI(1)
	}
	return true
}

func printHelp() {
	writeln(stdout, `CronTUI — Terminal UI for managing cron jobs

Usage:
  crontui              Launch interactive TUI
  crontui <command>    Run a CLI command

Commands:
  list, ls             List all cron jobs (--json for JSON output)
  add <sched> <cmd>    Add a new cron job (--desc "description")
  delete, rm <id>      Delete a cron job by ID
  enable <id>          Enable a cron job
  disable <id>         Disable a cron job
  validate <expr>      Validate a cron expression
  preview <expr> [n]   Show next N runs for expression
  runnow, run <id>     Execute a cron job immediately
  backup               Create a managed jobs backup
  restore <file>       Restore jobs from backup
  export [--format=X]  Export jobs as json or crontab format
  import <file.json>   Import jobs from JSON file
  completion [shell]   Generate shell completions (bash, zsh, fish)
  help                 Show this help message
  version              Show version

Flags:
  --debug              Enable debug logging to stderr`)
}
