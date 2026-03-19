//go:build windows

package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestWindowsMetadataRoundTrip(t *testing.T) {
	if os.Getenv("CRONTUI_WINDOWS_E2E") != "1" {
		t.Skip("set CRONTUI_WINDOWS_E2E=1 to run Windows Task Scheduler integration tests")
	}

	taskPath := `\CronTUI-Test\`
	taskName := taskNameForID(42)
	taskSource := encodeTaskSource("0 9 * * 1-5")
	startBoundary := time.Now().Add(5 * time.Minute).Format("2006-01-02T15:04:05Z07:00")
	userSID, err := currentUserSID()
	if err != nil {
		t.Fatalf("currentUserSID: %v", err)
	}

	xml := renderMetadataProbeTaskXML(taskPath, taskName, taskSource, "metadata round-trip", startBoundary, userSID)

	if out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf(`Register-ScheduledTask -TaskName '%s' -TaskPath '%s' -Xml @'
%s
'@ -User $env:USERNAME -Force | Out-Null`, taskName, taskPath, xml),
	).CombinedOutput(); err != nil {
		t.Fatalf("Register-ScheduledTask failed: %v\n%s", err, out)
	}
	defer func() {
		out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
			fmt.Sprintf(`Unregister-ScheduledTask -TaskName '%s' -TaskPath '%s' -Confirm:$false -ErrorAction SilentlyContinue | Out-Null`, taskName, taskPath),
		).CombinedOutput()
		if err != nil {
			t.Fatalf("Unregister-ScheduledTask cleanup failed: %v\n%s", err, out)
		}
	}()

	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf(`Export-ScheduledTask -TaskName '%s' -TaskPath '%s'`, taskName, taskPath),
	).CombinedOutput()
	if err != nil {
		t.Fatalf("Export-ScheduledTask failed: %v\n%s", err, out)
	}

	exported := string(out)
	for _, want := range []string{
		"<Source>" + taskSource + "</Source>",
		"<Description>metadata round-trip</Description>",
		"<URI>" + taskPath + taskName + "</URI>",
	} {
		if !strings.Contains(exported, want) {
			t.Fatalf("exported XML missing %q\nfull xml:\n%s", want, exported)
		}
	}
}

func currentUserSID() (string, error) {
	out, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command",
		"[System.Security.Principal.WindowsIdentity]::GetCurrent().User.Value",
	).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("resolve current user sid: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func renderMetadataProbeTaskXML(taskPath, taskName, taskSource, description, startBoundary, userSID string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-16"?>
<Task version="1.2" xmlns="http://schemas.microsoft.com/windows/2004/02/mit/task">
  <RegistrationInfo>
    <Source>%s</Source>
    <Description>%s</Description>
    <URI>%s%s</URI>
  </RegistrationInfo>
  <Principals>
    <Principal id="Author">
      <UserId>%s</UserId>
      <LogonType>InteractiveToken</LogonType>
    </Principal>
  </Principals>
  <Settings>
    <DisallowStartIfOnBatteries>true</DisallowStartIfOnBatteries>
    <StopIfGoingOnBatteries>true</StopIfGoingOnBatteries>
    <MultipleInstancesPolicy>IgnoreNew</MultipleInstancesPolicy>
    <IdleSettings>
      <Duration>PT10M</Duration>
      <WaitTimeout>PT1H</WaitTimeout>
      <StopOnIdleEnd>true</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
  </Settings>
  <Triggers>
    <TimeTrigger>
      <StartBoundary>%s</StartBoundary>
    </TimeTrigger>
  </Triggers>
  <Actions Context="Author">
    <Exec>
      <Command>powershell.exe</Command>
      <Arguments>-NoProfile -Command "Write-Output probe"</Arguments>
    </Exec>
  </Actions>
</Task>`, taskSource, description, taskPath, taskName, userSID, startBoundary)
}
