package crontab

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/meru143/crontui/pkg/types"
)

// WriteCrontab writes the given jobs back to the system crontab.
// It creates a full crontab string from the jobs and pipes it to `crontab -`.
func WriteCrontab(jobs []types.CronJob) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("crontab is not supported on Windows")
	}

	content := FormatCrontab(jobs)

	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(content)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to write crontab: %w (%s)", err, string(out))
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
