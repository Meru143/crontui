package crontab

import (
	"strings"

	"github.com/meru143/crontui/pkg/types"
)

// ParseCrontab parses raw crontab content into CronJob structs.
// It handles:
//   - Standard 5-field cron lines
//   - Commented-out cron lines (disabled jobs)
//   - Description annotations (# description: ...)
//   - Working directory annotations (# workingdir: ...)
//   - Mailto annotations (# mailto: ...)
//   - Environment variable lines (VAR=value)
//   - Blank lines and standalone comments
func ParseCrontab(raw string) []types.CronJob {
	doc, err := ParseDocument(raw)
	if err != nil {
		return nil
	}
	return doc.Jobs()
}

// isEnvVar checks if a line looks like an environment variable assignment.
func isEnvVar(line string) bool {
	// Must contain = and start with a letter
	if !strings.Contains(line, "=") {
		return false
	}
	if len(line) == 0 {
		return false
	}
	first := line[0]
	return (first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_'
}

// isCronLine checks if a line starts with a valid cron time field.
func isCronLine(line string) bool {
	if len(line) == 0 {
		return false
	}
	first := line[0]
	return first == '*' || (first >= '0' && first <= '9')
}

// splitCronLine splits a 5-field cron line into schedule and command.
func splitCronLine(line string) (string, string) {
	fields := strings.Fields(line)
	if len(fields) < 6 {
		return line, ""
	}
	schedule := strings.Join(fields[:5], " ")
	command := strings.Join(fields[5:], " ")
	return schedule, command
}

// splitAtLine splits an @ shortcut line into schedule and command.
func splitAtLine(line string) (string, string) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return line, ""
	}
	return fields[0], strings.Join(fields[1:], " ")
}
