# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-03-19

### Added
- Initial project setup with Bubble Tea TUI framework
- Crontab reader/parser/writer with backup support
- Cron expression validator and next-run preview
- Interactive job list with search, sort, and filter
- Add/edit form with live validation and schedule preview
- Backup management with create/restore/prune
- CLI subcommands: list, add, delete, enable, disable, validate, preview, backup, restore, export, import, runnow

### Changed
- Preserve unknown crontab content during parsing and rewrites instead of rebuilding from jobs alone
- Route mutating CLI and TUI writes through backup-aware crontab helpers
- Make backup restores write raw backup content back to the system crontab losslessly

### Fixed
- Correct leading `--debug` argument parsing for CLI subcommands
- Reject invalid preview counts instead of panicking on nonpositive values
- Stop showing demo jobs on non-Windows crontab read failures
- Remove misleading Working Directory and Mailto editing from the TUI form
