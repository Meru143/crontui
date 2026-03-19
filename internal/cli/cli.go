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
		fmt.Fprintf(stderr, "Error loading config: %v\n", err)
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
		fmt.Println(version.Full())
		return true
	default:
		fmt.Fprintf(stderr, "Unknown command: %s\n\n", cmd)
		printHelp()
		exitCLI(1)
		return true
	}
}

func runList(cfg config.Config, args []string) bool {
	jobs, err := cliBackendFn(cfg).LoadJobs()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		exitCLI(1)
	}

	// Check for --json flag
	for _, a := range args {
		if a == "--json" {
			data, _ := json.MarshalIndent(jobs, "", "  ")
			fmt.Fprintln(stdout, string(data))
			return true
		}
	}

	w := tabwriter.NewWriter(stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tStatus\tSchedule\tCommand\tDescription")
	fmt.Fprintln(w, "--\t------\t--------\t-------\t-----------")

	for _, job := range jobs {
		status := "ON "
		if !job.Enabled {
			status = "OFF"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", job.ID, status, job.Schedule, job.Command, job.Description)
	}
	w.Flush()
	return true
}

func runAdd(cfg config.Config, args []string) bool {
	if len(args) < 2 {
		fmt.Fprintf(stderr, "Usage: crontui add <schedule> <command> [--desc \"description\"]\n")
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
		fmt.Fprintf(stderr, "Invalid schedule: %v\n", err)
		exitCLI(1)
	}
	if err := backend.ValidateManagedSchedule(schedule); err != nil {
		fmt.Fprintf(stderr, "Unsupported schedule on this platform: %v\n", err)
		exitCLI(1)
	}

	jobs, err := backend.LoadJobs()
	if err != nil {
		fmt.Fprintf(stderr, "Error loading jobs: %v\n", err)
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
		fmt.Fprintf(stderr, "Error saving jobs: %v\n", err)
		exitCLI(1)
	}

	fmt.Fprintf(stdout, "Added job #%d: %s %s\n", maxID+1, schedule, command)
	return true
}

func runDelete(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(stderr, "Usage: crontui delete <job-id>\n")
		exitCLI(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "Invalid job ID: %s\n", args[0])
		exitCLI(1)
	}

	backend := cliBackendFn(cfg)
	jobs, err := backend.LoadJobs()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
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
		fmt.Fprintf(stderr, "Job #%d not found\n", id)
		exitCLI(1)
	}

	if err := backend.SaveJobs(cfg, newJobs); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		exitCLI(1)
	}

	fmt.Fprintf(stdout, "Deleted job #%d\n", id)
	return true
}

func runToggle(cfg config.Config, args []string, enable bool) bool {
	if len(args) < 1 {
		action := "enable"
		if !enable {
			action = "disable"
		}
		fmt.Fprintf(stderr, "Usage: crontui %s <job-id>\n", action)
		exitCLI(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "Invalid job ID: %s\n", args[0])
		exitCLI(1)
	}

	backend := cliBackendFn(cfg)
	jobs, err := backend.LoadJobs()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
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
		fmt.Fprintf(stderr, "Job #%d not found\n", id)
		exitCLI(1)
	}

	if err := backend.SaveJobs(cfg, jobs); err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		exitCLI(1)
	}

	action := "enabled"
	if !enable {
		action = "disabled"
	}
	fmt.Fprintf(stdout, "Job #%d %s\n", id, action)
	return true
}

func runValidate(args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(stderr, "Usage: crontui validate <expression>\n")
		exitCLI(1)
	}

	expr := strings.Join(args, " ")
	valid, err := cron.Validate(expr)
	if valid {
		fmt.Fprintf(stdout, "✓ Valid: %s\n", expr)

		// Show next runs
		if notice, ok := cron.PreviewNotice(expr); ok {
			fmt.Fprintf(stdout, "\nNext runs:\n  %s\n", notice)
			return true
		}

		runs, _ := cron.NextRuns(expr, 5)
		if len(runs) > 0 {
			fmt.Fprintln(stdout, "\nNext runs:")
			for i, t := range runs {
				fmt.Fprintf(stdout, "  %d. %s  (%s)\n", i+1, t.Format("Mon Jan 2 15:04:05 2006"), cron.HumanReadable(t))
			}
		}
	} else {
		fmt.Fprintf(stderr, "✗ Invalid: %v\n", err)
		exitCLI(1)
	}
	return true
}

func runPreview(args []string) bool {
	expr, count, err := parsePreviewArgs(args)
	if err != nil {
		if len(args) < 1 {
			fmt.Fprintf(stderr, "Usage: crontui preview <expression> [count]\n")
		} else {
			fmt.Fprintf(stderr, "Invalid preview arguments: %v\n", err)
		}
		exitCLI(1)
	}

	if notice, ok := cron.PreviewNotice(expr); ok {
		fmt.Fprintf(stdout, "Next %d runs for: %s\n\n", count, expr)
		fmt.Fprintf(stdout, "  %s\n", notice)
		return true
	}

	runs, err := cron.NextRuns(expr, count)
	if err != nil {
		fmt.Fprintf(stderr, "Invalid expression: %v\n", err)
		exitCLI(1)
	}

	fmt.Fprintf(stdout, "Next %d runs for: %s\n\n", count, expr)
	for i, t := range runs {
		fmt.Fprintf(stdout, "  %d. %s  (%s)\n", i+1, t.Format("Mon Jan 2 15:04:05 2006"), cron.HumanReadable(t))
	}
	return true
}

func runBackup(cfg config.Config) bool {
	path, err := cliBackendFn(cfg).CreateBackup(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "Backup failed: %v\n", err)
		exitCLI(1)
	}
	fmt.Fprintf(stdout, "Backup created: %s\n", path)
	return true
}

func runRestore(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(stderr, "Usage: crontui restore <filename>\n")
		exitCLI(1)
	}
	if err := cliBackendFn(cfg).RestoreBackup(cfg, args[0]); err != nil {
		fmt.Fprintf(stderr, "Restore failed: %v\n", err)
		exitCLI(1)
	}
	fmt.Fprintf(stdout, "Restored from %s\n", args[0])
	return true
}

func runExport(cfg config.Config, args []string) bool {
	jobs, err := cliBackendFn(cfg).LoadJobs()
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
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
		fmt.Fprintln(stdout, string(data))
	case "crontab":
		fmt.Fprint(stdout, crontab.FormatCrontab(jobs))
	default:
		fmt.Fprintf(stderr, "Unknown format: %s\n", format)
		exitCLI(1)
	}
	return true
}

func runImport(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(stderr, "Usage: crontui import <file.json>\n")
		exitCLI(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "Error reading file: %v\n", err)
		exitCLI(1)
	}

	var jobs []types.CronJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		fmt.Fprintf(stderr, "Error parsing JSON: %v\n", err)
		exitCLI(1)
	}

	backend := cliBackendFn(cfg)
	for _, job := range jobs {
		valid, err := cron.Validate(job.Schedule)
		if !valid || err != nil {
			fmt.Fprintf(stderr, "Invalid schedule for job #%d: %v\n", job.ID, err)
			exitCLI(1)
		}
		if err := backend.ValidateManagedSchedule(job.Schedule); err != nil {
			fmt.Fprintf(stderr, "Unsupported schedule for job #%d on this platform: %v\n", job.ID, err)
			exitCLI(1)
		}
	}

	if err := backend.SaveJobs(cfg, jobs); err != nil {
		fmt.Fprintf(stderr, "Error writing jobs: %v\n", err)
		exitCLI(1)
	}

	fmt.Fprintf(stdout, "Imported %d jobs\n", len(jobs))
	return true
}

func runNow(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(stderr, "Usage: crontui runnow <job-id>\n")
		exitCLI(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "Invalid job ID: %s\n", args[0])
		exitCLI(1)
	}

	fmt.Fprintf(stdout, "Running job #%d\n\n", id)

	out, err := cliBackendFn(cfg).RunNow(id)
	if len(out) > 0 {
		fmt.Fprint(stdout, string(out))
		if !strings.HasSuffix(string(out), "\n") {
			fmt.Fprintln(stdout)
		}
	}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			fmt.Fprintf(stdout, "\nJob exited with code %d\n", exitErr.ExitCode())
		} else {
			fmt.Fprintf(stderr, "\nExecution error: %v\n", err)
		}
		exitCLI(1)
	}

	if len(out) == 0 {
		fmt.Fprintln(stdout, "(no output)")
	}
	fmt.Fprintln(stdout, "\nJob completed successfully")
	return true
}

func runCompletion(args []string) bool {
	shell := "bash"
	if len(args) > 0 {
		shell = args[0]
	}

	switch shell {
	case "bash":
		fmt.Fprint(stdout, completion.Bash())
	case "zsh":
		fmt.Fprint(stdout, completion.Zsh())
	case "fish":
		fmt.Fprint(stdout, completion.Fish())
	default:
		fmt.Fprintf(stderr, "Unsupported shell: %s (supported: bash, zsh, fish)\n", shell)
		exitCLI(1)
	}
	return true
}

func printHelp() {
	fmt.Fprintln(stdout, `CronTUI — Terminal UI for managing cron jobs

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
  backup               Create a crontab backup
  restore <file>       Restore crontab from backup
  export [--format=X]  Export jobs as json or crontab format
  import <file.json>   Import jobs from JSON file
  completion [shell]   Generate shell completions (bash, zsh, fish)
  help                 Show this help message
  version              Show version

Flags:
  --debug              Enable debug logging to stderr`)
}
