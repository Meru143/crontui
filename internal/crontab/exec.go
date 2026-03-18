package crontab

import (
	"fmt"
	"os/exec"
	"strings"
)

// ValidateCommand checks if a command contains potentially dangerous characters.
// Returns an error if the command looks suspicious - this is intentionally
// conservative. Users can run commands with --unsafe flag in CLI if needed.
func ValidateCommand(cmd string) error {
	dangerous := []string{";", "&&", "||", "|", "`", "$", ">>", ">", "<", "&", "\\", "$(", "${"}
	for _, char := range dangerous {
		if strings.Contains(cmd, char) {
			return fmt.Errorf("command contains potentially dangerous character '%s'", char)
		}
	}
	return nil
}

// ExecCommand runs a command safely by passing it directly to the shell.
// Note: This still uses shell execution for compatibility with cron commands.
// Call ValidateCommand first for untrusted input.
func ExecCommand(cmd string) *exec.Cmd {
	return exec.Command("sh", "-c", cmd)
}
