# CronTUI

A beautiful terminal UI for managing cron jobs, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-green)

## Features

- **Interactive TUI** — browse, add, edit, delete, and toggle cron jobs visually
- **Live validation** — cron expressions are validated in real-time as you type
- **Next-run preview** — see upcoming execution times before saving
- **Schedule presets** — quick-pick common schedules (hourly, daily, weekly, etc.)
- **Search & filter** — find jobs by command or filter by enabled/disabled
- **Backup & restore** — automatic backups before every write, with restore support
- **CLI mode** — every operation is also available as a non-interactive subcommand
- **Export / Import** — export jobs as JSON or crontab format, import from JSON

## Installation

### From source

```bash
go install github.com/meru143/crontui@latest
```

### Build locally

```bash
git clone https://github.com/meru143/crontui.git
cd crontui
make build
```

## Usage

### Interactive TUI

```bash
crontui
```

### CLI subcommands

```bash
crontui list                          # List all cron jobs
crontui list --json                   # List as JSON
crontui add "*/5 * * * *" "/usr/bin/backup.sh" --desc "Backup every 5 min"
crontui delete 3                      # Delete job #3
crontui enable 2                      # Enable job #2
crontui disable 2                     # Disable job #2
crontui validate "0 */2 * * *"        # Validate a cron expression
crontui preview "0 9 * * 1-5" 5       # Show next 5 runs
crontui backup                        # Create a backup
crontui restore <filename>            # Restore from backup
crontui export --format=json          # Export as JSON (default)
crontui export --format=crontab       # Export as raw crontab
crontui import jobs.json              # Import jobs from JSON
crontui version                       # Show version
crontui help                          # Show help
```

## Keyboard Shortcuts

### List View

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Home` / `g` | Jump to first job |
| `End` / `G` | Jump to last job |
| `a` | Add new job |
| `e` / `Enter` | Edit selected job |
| `d` | Delete selected job |
| `t` | Toggle enabled/disabled |
| `/` | Search jobs |
| `f` | Cycle filter (all → enabled → disabled) |
| `b` | Open backup list |
| `r` | Refresh job list |
| `q` | Quit |

### Form View (Add/Edit)

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Move between fields |
| `1`–`6` | Apply schedule preset |
| `Ctrl+S` | Save job |
| `Esc` | Cancel and return to list |

### Backup View

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate backups |
| `Enter` | Restore selected backup |
| `Esc` | Return to list |

## Project Structure

```
crontui/
├── main.go                  # Entry point: CLI dispatch or TUI launch
├── internal/
│   ├── cli/cli.go           # CLI subcommand handler
│   ├── config/config.go     # Configuration (Viper)
│   ├── cron/
│   │   ├── validator.go     # Cron expression validation (robfig/cron)
│   │   └── preview.go       # Next-run calculation & formatting
│   ├── crontab/
│   │   ├── reader.go        # Read system crontab
│   │   ├── writer.go        # Write system crontab
│   │   ├── parser.go        # Parse crontab lines into CronJob structs
│   │   └── backup.go        # Backup create/restore/prune/list
│   ├── model/
│   │   ├── model.go         # Bubble Tea model (Init/Update/View)
│   │   ├── list.go          # List view rendering & key handling
│   │   ├── form.go          # Add/edit form view
│   │   ├── backup.go        # Backup list view
│   │   └── helpers.go       # Utility functions
│   └── styles/styles.go     # Lip Gloss styles & color palette
└── pkg/types/cronjob.go     # CronJob struct
```

## Configuration

CronTUI stores backups in `~/.config/crontui/backups/`. The default configuration can be customized via environment variables with the `CRONTUI_` prefix.

## License

[MIT](LICENSE)
