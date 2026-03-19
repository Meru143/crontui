package scheduler

import (
	"os/exec"
)

var runImmediateCommand = defaultRunImmediateCommand

func defaultRunImmediateCommand(goos, command string) ([]byte, error) {
	var cmd *exec.Cmd
	if goos == "windows" {
		cmd = exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	return cmd.CombinedOutput()
}
