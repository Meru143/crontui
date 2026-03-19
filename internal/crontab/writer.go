package crontab

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

var (
	execCommand       = exec.Command
	writeRawCrontabFn = WriteRawCrontab
)

func normalizeCrontab(raw string) string {
	if strings.HasSuffix(raw, "\n") {
		return raw
	}
	return raw + "\n"
}

// LoadDocument reads and parses the current crontab into a document.
func LoadDocument() (*Document, error) {
	raw, err := ReadCrontab()
	if err != nil {
		return nil, err
	}
	return ParseDocument(raw)
}

// WriteCrontab writes the given jobs back to the system crontab.
// It creates a full crontab string from the jobs and pipes it to `crontab -`.
func WriteCrontab(jobs []types.CronJob) error {
	content := FormatCrontab(jobs)
	return WriteRawCrontab(content)
}

// WriteRawCrontab writes raw crontab text directly to the system crontab.
func WriteRawCrontab(raw string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("crontab is not supported on Windows")
	}

	cmd := execCommand("crontab", "-")
	cmd.Stdin = strings.NewReader(normalizeCrontab(raw))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to write crontab: %w (%s)", err, string(out))
	}
	return nil
}

// WriteDocument writes a parsed crontab document back to the system crontab.
func WriteDocument(doc *Document) error {
	if doc == nil {
		return fmt.Errorf("document is required")
	}
	return writeRawCrontabFn(normalizeCrontab(doc.Render()))
}

// WriteDocumentWithBackup creates a backup, writes the document, then prunes old backups.
func WriteDocumentWithBackup(cfg config.Config, doc *Document) error {
	if doc == nil {
		return fmt.Errorf("document is required")
	}
	if _, err := createBackupFn(cfg); err != nil {
		return err
	}
	if err := writeRawCrontabFn(normalizeCrontab(doc.Render())); err != nil {
		return err
	}
	if err := PruneBackups(cfg); err != nil {
		return err
	}
	return nil
}

// WriteJobsWithBackup replaces the editable jobs while preserving raw non-job lines.
func WriteJobsWithBackup(cfg config.Config, jobs []types.CronJob) error {
	doc, err := LoadDocument()
	if err != nil {
		return err
	}
	if err := doc.ReplaceJobs(jobs); err != nil {
		return err
	}
	return WriteDocumentWithBackup(cfg, doc)
}

// RemoveCrontab removes the current user's crontab entirely (crontab -r).
func RemoveCrontab() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("crontab is not supported on Windows")
	}

	cmd := execCommand("crontab", "-r")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to remove crontab: %w (%s)", err, string(out))
	}
	return nil
}

// RemoveCrontabWithBackup backs up the current crontab before removing it entirely.
func RemoveCrontabWithBackup(cfg config.Config) error {
	if _, err := createBackupFn(cfg); err != nil {
		return err
	}
	if err := RemoveCrontab(); err != nil {
		return err
	}
	if cfg.MaxBackups > 0 {
		if err := PruneBackups(cfg); err != nil {
			return err
		}
	}
	return nil
}

// FormatCrontab formats jobs into crontab file content.
func FormatCrontab(jobs []types.CronJob) string {
	var b strings.Builder
	b.WriteString("# Crontab managed by crontui\n")
	b.WriteString("# Do not edit the marker comments\n\n")

	for _, job := range jobs {
		if job.Description != "" {
			b.WriteString(fmt.Sprintf("# description: %s\n", job.Description))
		}

		line := fmt.Sprintf("%s %s", job.Schedule, job.Command)
		if !job.Enabled {
			line = "#" + line
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}
