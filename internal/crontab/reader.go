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
		output := string(out)
		// "no crontab for <user>" is not an error — it means empty
		if strings.Contains(output, "no crontab for") {
			return "", nil
		}
		return "", fmt.Errorf("failed to read crontab: %w (%s)", err, output)
	}
	return string(out), nil
}
