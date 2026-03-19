package scheduler

import (
	"fmt"
	"os/exec"
)

type PowerShellRunner interface {
	Run(script string) ([]byte, error)
}

type commandPowerShellRunner struct{}

func newPowerShellRunner() PowerShellRunner {
	return commandPowerShellRunner{}
}

func (commandPowerShellRunner) Run(script string) ([]byte, error) {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("powershell command failed: %w (%s)", err, out)
	}
	return out, nil
}
