package crontab

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ReadCrontab reads the current user's crontab.
// Returns the raw crontab content as a string.
// Returns empty string (not error) when no crontab exists.
func ReadCrontab() (string, error) {
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("crontab is not supported on Windows")
	}

	cmd := exec.Command("crontab", "-l")
	out, err := cmd.CombinedOutput()
	if err != nil {
		output := strings.TrimSpace(string(out))
		// "no crontab for <user>" is not an error — it means empty
		if strings.Contains(output, "no crontab for") {
			return "", nil
		}
		// 12.2: Permission denied — suggest chmod
		if strings.Contains(output, "Permission denied") || strings.Contains(output, "permission denied") {
			return "", fmt.Errorf("permission denied reading crontab. Try: sudo chmod 644 /var/spool/cron/crontabs/$USER")
		}
		// 12.1: crontab binary not found
		if strings.Contains(err.Error(), "executable file not found") || strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("crontab command not found. Install it with: sudo apt install cron (Debian/Ubuntu) or sudo yum install cronie (RHEL/CentOS)")
		}
		return "", fmt.Errorf("failed to read crontab: %w (%s)", err, output)
	}
	return string(out), nil
}
