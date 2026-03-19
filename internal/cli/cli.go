package cli

import (
	"encoding/json"
	"fmt"
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
	"github.com/meru143/crontui/internal/version"
	"github.com/meru143/crontui/pkg/types"
)

// Debug controls verbose logging output.
var Debug bool

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
		log.SetOutput(os.Stderr)
		log.SetPrefix("[debug] ")
		log.Println("debug mode enabled")
	}

	cfg := config.DefaultConfig()

	switch cmd {
	case "list", "ls":
		return runList(subArgs)
	case "add":
		return runAdd(subArgs)
	case "delete", "rm":
		return runDelete(subArgs)
	case "enable":
		return runToggle(subArgs, true)
	case "disable":
		return runToggle(subArgs, false)
	case "validate":
		return runValidate(subArgs)
	case "preview":
		return runPreview(subArgs)
	case "backup":
		return runBackup(cfg)
	case "restore":
		return runRestore(cfg, subArgs)
	case "export":
		return runExport(subArgs)
	case "import":
		return runImport(subArgs)
	case "runnow", "run":
		return runNow(subArgs)
	case "completion", "--completion":
		return runCompletion(subArgs)
	case "help", "--help", "-h":
		printHelp()
		return true
	case "version", "--version", "-v":
		fmt.Println(version.Full())
		return true
	default:
		return false
	}
}

func runList(args []string) bool {
	raw, err := crontab.ReadCrontab()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	jobs := crontab.ParseCrontab(raw)

	// Check for --json flag
	for _, a := range args {
		if a == "--json" {
			data, _ := json.MarshalIndent(jobs, "", "  ")
			fmt.Println(string(data))
			return true
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
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

func runAdd(args []string) bool {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: crontui add <schedule> <command> [--desc \"description\"]\n")
		os.Exit(1)
	}

	schedule := args[0]
	command := args[1]
	description := ""

	for i, a := range args {
		if a == "--desc" && i+1 < len(args) {
			description = args[i+1]
			break
		}
	}

	// Validate schedule
	valid, err := cron.Validate(schedule)
	if !valid {
		fmt.Fprintf(os.Stderr, "Invalid schedule: %v\n", err)
		os.Exit(1)
	}

	raw, err := crontab.ReadCrontab()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading crontab: %v\n", err)
		os.Exit(1)
	}

	jobs := crontab.ParseCrontab(raw)
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

	if err := crontab.WriteCrontab(jobs); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing crontab: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Added job #%d: %s %s\n", maxID+1, schedule, command)
	return true
}

func runDelete(args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: crontui delete <job-id>\n")
		os.Exit(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid job ID: %s\n", args[0])
		os.Exit(1)
	}

	raw, err := crontab.ReadCrontab()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	jobs := crontab.ParseCrontab(raw)
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
		fmt.Fprintf(os.Stderr, "Job #%d not found\n", id)
		os.Exit(1)
	}

	if err := crontab.WriteCrontab(newJobs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Deleted job #%d\n", id)
	return true
}

func runToggle(args []string, enable bool) bool {
	if len(args) < 1 {
		action := "enable"
		if !enable {
			action = "disable"
		}
		fmt.Fprintf(os.Stderr, "Usage: crontui %s <job-id>\n", action)
		os.Exit(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid job ID: %s\n", args[0])
		os.Exit(1)
	}

	raw, err := crontab.ReadCrontab()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	jobs := crontab.ParseCrontab(raw)
	found := false
	for i := range jobs {
		if jobs[i].ID == id {
			jobs[i].Enabled = enable
			found = true
			break
		}
	}

	if !found {
		fmt.Fprintf(os.Stderr, "Job #%d not found\n", id)
		os.Exit(1)
	}

	if err := crontab.WriteCrontab(jobs); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	action := "enabled"
	if !enable {
		action = "disabled"
	}
	fmt.Printf("Job #%d %s\n", id, action)
	return true
}

func runValidate(args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: crontui validate <expression>\n")
		os.Exit(1)
	}

	expr := strings.Join(args, " ")
	valid, err := cron.Validate(expr)
	if valid {
		fmt.Printf("✓ Valid: %s\n", expr)

		// Show next runs
		runs, _ := cron.NextRuns(expr, 5)
		if len(runs) > 0 {
			fmt.Println("\nNext runs:")
			for i, t := range runs {
				fmt.Printf("  %d. %s  (%s)\n", i+1, t.Format("Mon Jan 2 15:04:05 2006"), cron.HumanReadable(t))
			}
		}
	} else {
		fmt.Fprintf(os.Stderr, "✗ Invalid: %v\n", err)
		os.Exit(1)
	}
	return true
}

func runPreview(args []string) bool {
	expr, count, err := parsePreviewArgs(args)
	if err != nil {
		if len(args) < 1 {
			fmt.Fprintf(os.Stderr, "Usage: crontui preview <expression> [count]\n")
		} else {
			fmt.Fprintf(os.Stderr, "Invalid preview arguments: %v\n", err)
		}
		os.Exit(1)
	}

	runs, err := cron.NextRuns(expr, count)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid expression: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Next %d runs for: %s\n\n", count, expr)
	for i, t := range runs {
		fmt.Printf("  %d. %s  (%s)\n", i+1, t.Format("Mon Jan 2 15:04:05 2006"), cron.HumanReadable(t))
	}
	return true
}

func runBackup(cfg config.Config) bool {
	path, err := crontab.CreateBackup(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Backup failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Backup created: %s\n", path)
	return true
}

func runRestore(cfg config.Config, args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: crontui restore <filename>\n")
		os.Exit(1)
	}
	if err := crontab.RestoreBackup(cfg, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Restore failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Restored from %s\n", args[0])
	return true
}

func runExport(args []string) bool {
	raw, err := crontab.ReadCrontab()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	jobs := crontab.ParseCrontab(raw)

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
		fmt.Println(string(data))
	case "crontab":
		fmt.Print(crontab.FormatCrontab(jobs))
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", format)
		os.Exit(1)
	}
	return true
}

func runImport(args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: crontui import <file.json>\n")
		os.Exit(1)
	}

	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}

	var jobs []types.CronJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	if err := crontab.WriteCrontab(jobs); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing crontab: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Imported %d jobs\n", len(jobs))
	return true
}

func runNow(args []string) bool {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: crontui runnow <job-id>\n")
		os.Exit(1)
	}

	id, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid job ID: %s\n", args[0])
		os.Exit(1)
	}

	raw, err := crontab.ReadCrontab()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	jobs := crontab.ParseCrontab(raw)
	var target *types.CronJob
	for i := range jobs {
		if jobs[i].ID == id {
			target = &jobs[i]
			break
		}
	}

	if target == nil {
		fmt.Fprintf(os.Stderr, "Job #%d not found\n", id)
		os.Exit(1)
	}

	fmt.Printf("Running job #%d: %s\n\n", id, target.Command)

	cmd := crontab.ExecCommand(target.Command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Printf("\nJob exited with code %d\n", exitErr.ExitCode())
		} else {
			fmt.Fprintf(os.Stderr, "\nExecution error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("\nJob completed successfully")
	return true
}

func runCompletion(args []string) bool {
	shell := "bash"
	if len(args) > 0 {
		shell = args[0]
	}

	switch shell {
	case "bash":
		fmt.Print(completion.Bash())
	case "zsh":
		fmt.Print(completion.Zsh())
	case "fish":
		fmt.Print(completion.Fish())
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s (supported: bash, zsh, fish)\n", shell)
		os.Exit(1)
	}
	return true
}

func printHelp() {
	fmt.Println(`CronTUI — Terminal UI for managing cron jobs

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
