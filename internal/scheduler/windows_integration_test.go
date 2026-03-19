//go:build windows

package scheduler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
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

func TestWindowsCRUDRoundTrip(t *testing.T) {
	if os.Getenv("CRONTUI_WINDOWS_E2E") != "1" {
		t.Skip("set CRONTUI_WINDOWS_E2E=1 to run Windows Task Scheduler integration tests")
	}

	cfg := config.DefaultConfig()
	cfg.WindowsTaskPath = `\CronTUI-Test\`

	backend := &windowsBackend{
		cfg:    cfg,
		runner: newPowerShellRunner(),
	}

	if err := backend.SaveJobs(cfg, nil); err != nil {
		t.Fatalf("initial cleanup failed: %v", err)
	}
	defer func() {
		if err := backend.SaveJobs(cfg, nil); err != nil {
			t.Fatalf("final cleanup failed: %v", err)
		}
	}()

	initial := []types.CronJob{
		{ID: 1, Schedule: "0 9 * * 1-5", Command: `Write-Output "one"`, Description: "weekday", Enabled: true},
		{ID: 3, Schedule: "0 0 1 * *", Command: `Write-Output "three"`, Description: "monthly", Enabled: false},
	}
	if err := backend.SaveJobs(cfg, initial); err != nil {
		t.Fatalf("SaveJobs(initial) failed: %v", err)
	}

	loaded, err := backend.LoadJobs()
	if err != nil {
		t.Fatalf("LoadJobs(initial) failed: %v", err)
	}
	if !reflect.DeepEqual(loaded, initial) {
		t.Fatalf("loaded initial jobs = %#v, want %#v", loaded, initial)
	}

	updated := []types.CronJob{
		{ID: 1, Schedule: "30 2 * * *", Command: `Write-Output "updated"`, Description: "nightly", Enabled: true},
		{ID: 2, Schedule: "0 0 1 1 *", Command: `Write-Output "two"`, Description: "yearly", Enabled: false},
	}
	if err := backend.SaveJobs(cfg, updated); err != nil {
		t.Fatalf("SaveJobs(updated) failed: %v", err)
	}

	loaded, err = backend.LoadJobs()
	if err != nil {
		t.Fatalf("LoadJobs(updated) failed: %v", err)
	}
	if !reflect.DeepEqual(loaded, updated) {
		t.Fatalf("loaded updated jobs = %#v, want %#v", loaded, updated)
	}
}

func TestWindowsBackupRunRestoreAndRemoveAll(t *testing.T) {
	if os.Getenv("CRONTUI_WINDOWS_E2E") != "1" {
		t.Skip("set CRONTUI_WINDOWS_E2E=1 to run Windows Task Scheduler integration tests")
	}

	cfg := config.DefaultConfig()
	cfg.BackupDir = t.TempDir()
	cfg.WindowsTaskPath = `\CronTUI-Test\`

	backend := &windowsBackend{
		cfg:    cfg,
		runner: newPowerShellRunner(),
	}

	if err := backend.SaveJobs(cfg, nil); err != nil {
		t.Fatalf("initial cleanup failed: %v", err)
	}
	defer func() {
		if err := backend.SaveJobs(cfg, nil); err != nil {
			t.Fatalf("final cleanup failed: %v", err)
		}
	}()

	initial := []types.CronJob{
		{ID: 1, Schedule: "30 2 * * *", Command: `Write-Output "integration-run"`, Description: "integration", Enabled: true},
	}
	if err := backend.SaveJobs(cfg, initial); err != nil {
		t.Fatalf("SaveJobs(initial) failed: %v", err)
	}

	out, err := backend.RunNow(1)
	if err != nil {
		t.Fatalf("RunNow failed: %v", err)
	}
	if !strings.Contains(string(out), "integration-run") {
		t.Fatalf("RunNow output = %q, want integration-run", string(out))
	}

	backupPath, err := backend.CreateBackup(cfg)
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file %q missing: %v", backupPath, err)
	}

	backups, err := backend.ListBackups(cfg)
	if err != nil {
		t.Fatalf("ListBackups failed: %v", err)
	}
	if len(backups) != 1 || backups[0].JobCount != 1 {
		t.Fatalf("backups = %#v, want one backup with one job", backups)
	}

	if err := backend.RemoveAll(cfg); err != nil {
		t.Fatalf("RemoveAll failed: %v", err)
	}
	loaded, err := backend.LoadJobs()
	if err != nil {
		t.Fatalf("LoadJobs(after RemoveAll) failed: %v", err)
	}
	if len(loaded) != 0 {
		t.Fatalf("LoadJobs(after RemoveAll) = %#v, want empty", loaded)
	}

	if err := backend.RestoreBackup(cfg, filepath.Base(backupPath)); err != nil {
		t.Fatalf("RestoreBackup failed: %v", err)
	}
	loaded, err = backend.LoadJobs()
	if err != nil {
		t.Fatalf("LoadJobs(after RestoreBackup) failed: %v", err)
	}
	if !reflect.DeepEqual(loaded, initial) {
		t.Fatalf("LoadJobs(after RestoreBackup) = %#v, want %#v", loaded, initial)
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
