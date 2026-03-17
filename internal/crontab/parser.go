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
//   - Environment variable lines (VAR=value)
//   - Blank lines and standalone comments
func ParseCrontab(raw string) []types.CronJob {
	lines := strings.Split(raw, "\n")
	var jobs []types.CronJob
	id := 1
	pendingDescription := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip blank lines
		if trimmed == "" {
			pendingDescription = ""
			continue
		}

		// Check for description annotation: # description: some text
		if strings.HasPrefix(trimmed, "# description:") {
			pendingDescription = strings.TrimSpace(strings.TrimPrefix(trimmed, "# description:"))
			continue
		}

		// Skip environment variable lines (e.g. SHELL=/bin/bash)
		if isEnvVar(trimmed) {
			pendingDescription = ""
			continue
		}

		// Check for disabled cron job: commented-out cron line
		if strings.HasPrefix(trimmed, "#") {
			uncommented := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			if isCronLine(uncommented) {
				schedule, command := splitCronLine(uncommented)
				jobs = append(jobs, types.CronJob{
					ID:          id,
					Schedule:    schedule,
					Command:     command,
					Description: pendingDescription,
					Enabled:     false,
				})
				id++
				pendingDescription = ""
				continue
			}
			// Standalone comment — skip
			pendingDescription = ""
			continue
		}

		// Active cron line
		if isCronLine(trimmed) {
			schedule, command := splitCronLine(trimmed)
			jobs = append(jobs, types.CronJob{
				ID:          id,
				Schedule:    schedule,
				Command:     command,
				Description: pendingDescription,
				Enabled:     true,
			})
			id++
			pendingDescription = ""
			continue
		}

		// Handle @reboot, @hourly, etc. shortcuts
		if strings.HasPrefix(trimmed, "@") {
			schedule, command := splitAtLine(trimmed)
			if command != "" {
				jobs = append(jobs, types.CronJob{
					ID:          id,
					Schedule:    schedule,
					Command:     command,
					Description: pendingDescription,
					Enabled:     true,
				})
				id++
			}
			pendingDescription = ""
			continue
		}

		pendingDescription = ""
	}

	return jobs
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
