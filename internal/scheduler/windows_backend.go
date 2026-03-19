package scheduler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

type windowsBackend struct {
	cfg    config.Config
	runner PowerShellRunner
}

func (b *windowsBackend) LoadJobs() ([]types.CronJob, error) {
	records, err := b.listManagedTasks()
	if err != nil {
		return nil, err
	}

	jobs := make([]types.CronJob, 0, len(records))
	for _, record := range records {
		id, err := parseIDFromTaskName(record.TaskName)
		if err != nil {
			return nil, err
		}

		schedule, err := decodeTaskSource(record.Source)
		if err != nil {
			return nil, err
		}

		command, err := unwrapWindowsCommand(record.Execute, record.Arguments)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, types.CronJob{
			ID:          id,
			Schedule:    schedule,
			Command:     command,
			Description: record.Description,
			Enabled:     record.Enabled,
		})
	}

	slices.SortFunc(jobs, func(a, b types.CronJob) int {
		return a.ID - b.ID
	})

	return jobs, nil
}

func (b *windowsBackend) SaveJobs(cfg config.Config, jobs []types.CronJob) error {
	if b.runner == nil {
		b.runner = newPowerShellRunner()
	}

	current, err := b.listManagedTasks()
	if err != nil {
		return err
	}

	desiredNames := make(map[string]struct{}, len(jobs))
	for _, job := range jobs {
		spec, err := buildWindowsTaskSpec(cfg.WindowsTaskPath, job)
		if err != nil {
			return err
		}

		script, err := renderRegisterTaskScript(spec)
		if err != nil {
			return err
		}
		if _, err := b.runner.Run(script); err != nil {
			return err
		}

		desiredNames[spec.TaskName] = struct{}{}
	}

	for _, record := range current {
		if _, ok := desiredNames[record.TaskName]; ok {
			continue
		}
		if _, err := b.runner.Run(unregisterTaskScript(record.TaskPath, record.TaskName)); err != nil {
			return err
		}
	}

	return nil
}

func (b *windowsBackend) CreateBackup(config.Config) (string, error) {
	return "", errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) ListBackups(config.Config) ([]types.Backup, error) {
	return nil, errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) RestoreBackup(config.Config, string) error {
	return errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) RemoveAll(config.Config) error {
	return errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) RunNow(int) ([]byte, error) {
	return nil, errBackendNotImplemented("windows scheduler")
}

func (b *windowsBackend) ValidateManagedSchedule(expr string) error {
	_, err := translateWindowsSchedule(expr)
	return err
}

type windowsTaskRecord struct {
	TaskName    string `json:"TaskName"`
	TaskPath    string `json:"TaskPath"`
	Source      string `json:"Source"`
	Description string `json:"Description"`
	Enabled     bool   `json:"Enabled"`
	Execute     string `json:"Execute"`
	Arguments   string `json:"Arguments"`
}

func (b *windowsBackend) listManagedTasks() ([]windowsTaskRecord, error) {
	if b.runner == nil {
		b.runner = newPowerShellRunner()
	}

	out, err := b.runner.Run(listManagedTasksScript(b.cfg.WindowsTaskPath))
	if err != nil {
		return nil, err
	}

	return parseWindowsTaskRecords(out)
}

func listManagedTasksScript(taskPath string) string {
	return fmt.Sprintf(`$tasks = @(Get-ScheduledTask -TaskPath '%s' -ErrorAction SilentlyContinue)
$tasks | ForEach-Object {
  $action = $_.Actions | Select-Object -First 1
  [pscustomobject]@{
    TaskName = $_.TaskName
    TaskPath = $_.TaskPath
    Source = $_.Source
    Description = $_.Description
    Enabled = $_.Settings.Enabled
    Execute = if ($null -ne $action) { $action.Execute } else { $null }
    Arguments = if ($null -ne $action) { $action.Arguments } else { $null }
  }
} | ConvertTo-Json -Depth 4`, taskPath)
}

func parseWindowsTaskRecords(out []byte) ([]windowsTaskRecord, error) {
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	var records []windowsTaskRecord
	if err := json.Unmarshal(trimmed, &records); err == nil {
		return records, nil
	}

	var record windowsTaskRecord
	if err := json.Unmarshal(trimmed, &record); err != nil {
		return nil, fmt.Errorf("parse windows task list: %w", err)
	}
	return []windowsTaskRecord{record}, nil
}

func renderRegisterTaskScript(spec windowsTaskSpec) (string, error) {
	xmlContent, err := renderTaskXML(spec)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(`$xml = @'
%s
'@
Register-ScheduledTask -TaskName '%s' -TaskPath '%s' -Xml $xml -User $env:USERNAME -Force | Out-Null`, xmlContent, spec.TaskName, spec.TaskPath), nil
}

func unregisterTaskScript(taskPath, taskName string) string {
	return fmt.Sprintf(`Unregister-ScheduledTask -TaskName '%s' -TaskPath '%s' -Confirm:$false -ErrorAction SilentlyContinue | Out-Null`, taskName, taskPath)
}

func renderTaskXML(spec windowsTaskSpec) (string, error) {
	user := os.Getenv("USERNAME")
	if user == "" {
		return "", fmt.Errorf("USERNAME is required to register Windows scheduled tasks")
	}

	triggerXML, err := renderTriggerXML(spec.Triggers[0], time.Now())
	if err != nil {
		return "", err
	}

	execute, arguments := wrapWindowsCommand(spec.Command)
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
    <Enabled>%t</Enabled>
    <IdleSettings>
      <Duration>PT10M</Duration>
      <WaitTimeout>PT1H</WaitTimeout>
      <StopOnIdleEnd>true</StopOnIdleEnd>
      <RestartOnIdle>false</RestartOnIdle>
    </IdleSettings>
  </Settings>
  <Triggers>
%s
  </Triggers>
  <Actions Context="Author">
    <Exec>
      <Command>%s</Command>
      <Arguments>%s</Arguments>
    </Exec>
  </Actions>
</Task>`,
		xmlEscape(spec.Source),
		xmlEscape(spec.Description),
		xmlEscape(spec.TaskPath),
		xmlEscape(spec.TaskName),
		xmlEscape(user),
		spec.Enabled,
		triggerXML,
		xmlEscape(execute),
		xmlEscape(arguments),
	), nil
}

func renderTriggerXML(trigger windowsTriggerSpec, now time.Time) (string, error) {
	switch trigger.Type {
	case windowsTriggerDaily:
		startBoundary := nextDailyBoundary(trigger.AtHour, trigger.AtMinute, now).Format(time.RFC3339)
		var repetition string
		if trigger.EveryMinutes > 0 {
			repetition = fmt.Sprintf("\n      <Repetition>\n        <Interval>PT%dM</Interval>\n        <Duration>P1D</Duration>\n        <StopAtDurationEnd>false</StopAtDurationEnd>\n      </Repetition>", trigger.EveryMinutes)
		}
		return fmt.Sprintf("    <CalendarTrigger>\n      <StartBoundary>%s</StartBoundary>\n      <Enabled>true</Enabled>%s\n      <ScheduleByDay>\n        <DaysInterval>1</DaysInterval>\n      </ScheduleByDay>\n    </CalendarTrigger>", startBoundary, repetition), nil

	case windowsTriggerWeekly:
		startBoundary := nextWeeklyBoundary(trigger.Weekdays, trigger.AtHour, trigger.AtMinute, now)
		return fmt.Sprintf("    <CalendarTrigger>\n      <StartBoundary>%s</StartBoundary>\n      <Enabled>true</Enabled>\n      <ScheduleByWeek>\n        <WeeksInterval>1</WeeksInterval>\n        <DaysOfWeek>%s</DaysOfWeek>\n      </ScheduleByWeek>\n    </CalendarTrigger>", startBoundary.Format(time.RFC3339), renderWeekdaysXML(trigger.Weekdays)), nil

	case windowsTriggerMonthly:
		startBoundary, err := nextMonthlyBoundary(trigger.DayOfMonth, trigger.AtHour, trigger.AtMinute, now)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("    <CalendarTrigger>\n      <StartBoundary>%s</StartBoundary>\n      <Enabled>true</Enabled>\n      <ScheduleByMonth>\n        <DaysOfMonth><Day>%d</Day></DaysOfMonth>\n        <Months>%s</Months>\n      </ScheduleByMonth>\n    </CalendarTrigger>", startBoundary.Format(time.RFC3339), trigger.DayOfMonth, renderMonthsXML([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})), nil

	case windowsTriggerYearly:
		startBoundary, err := nextYearlyBoundary(trigger.Months[0], trigger.DayOfMonth, trigger.AtHour, trigger.AtMinute, now)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("    <CalendarTrigger>\n      <StartBoundary>%s</StartBoundary>\n      <Enabled>true</Enabled>\n      <ScheduleByMonth>\n        <DaysOfMonth><Day>%d</Day></DaysOfMonth>\n        <Months>%s</Months>\n      </ScheduleByMonth>\n    </CalendarTrigger>", startBoundary.Format(time.RFC3339), trigger.DayOfMonth, renderMonthsXML(trigger.Months)), nil
	}

	return "", fmt.Errorf("unsupported windows trigger type %q", trigger.Type)
}

func renderWeekdaysXML(weekdays []int) string {
	var b strings.Builder
	for _, day := range weekdays {
		fmt.Fprintf(&b, "<%s/>", weekdayElementName(day))
	}
	return b.String()
}

func renderMonthsXML(months []int) string {
	var b strings.Builder
	for _, month := range months {
		fmt.Fprintf(&b, "<%s/>", monthElementName(month))
	}
	return b.String()
}

func nextDailyBoundary(hour, minute int, now time.Time) time.Time {
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !candidate.After(now) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate
}

func nextWeeklyBoundary(weekdays []int, hour, minute int, now time.Time) time.Time {
	base := nextDailyBoundary(hour, minute, now)
	for offset := 0; offset < 8; offset++ {
		candidate := base.AddDate(0, 0, offset)
		if slices.Contains(weekdays, int(candidate.Weekday())) {
			return candidate
		}
	}
	return base
}

func nextMonthlyBoundary(day, hour, minute int, now time.Time) (time.Time, error) {
	for offset := 0; offset < 24; offset++ {
		candidateMonth := now.AddDate(0, offset, 0)
		candidate := time.Date(candidateMonth.Year(), candidateMonth.Month(), day, hour, minute, 0, 0, now.Location())
		if candidate.Month() != candidateMonth.Month() {
			continue
		}
		if candidate.After(now) {
			return candidate, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not compute monthly start boundary for day %d", day)
}

func nextYearlyBoundary(month, day, hour, minute int, now time.Time) (time.Time, error) {
	for year := now.Year(); year <= now.Year()+10; year++ {
		candidate := time.Date(year, time.Month(month), day, hour, minute, 0, 0, now.Location())
		if int(candidate.Month()) != month {
			continue
		}
		if candidate.After(now) {
			return candidate, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not compute yearly start boundary for month %d day %d", month, day)
}

func weekdayElementName(day int) string {
	switch day {
	case 0:
		return "Sunday"
	case 1:
		return "Monday"
	case 2:
		return "Tuesday"
	case 3:
		return "Wednesday"
	case 4:
		return "Thursday"
	case 5:
		return "Friday"
	default:
		return "Saturday"
	}
}

func monthElementName(month int) string {
	return [...]string{
		"",
		"January",
		"February",
		"March",
		"April",
		"May",
		"June",
		"July",
		"August",
		"September",
		"October",
		"November",
		"December",
	}[month]
}

func wrapWindowsCommand(command string) (execute string, arguments string) {
	return "powershell.exe", "-NoProfile -NonInteractive -EncodedCommand " + encodeUTF16LEBase64(command)
}

func unwrapWindowsCommand(execute, arguments string) (string, error) {
	if !strings.EqualFold(execute, "powershell.exe") {
		return "", fmt.Errorf("managed task execute %q is not supported", execute)
	}

	const prefix = "-NoProfile -NonInteractive -EncodedCommand "
	if !strings.HasPrefix(arguments, prefix) {
		return "", fmt.Errorf("managed task arguments %q are not in the expected encoded-command format", arguments)
	}

	decoded, err := decodeUTF16LEBase64(strings.TrimPrefix(arguments, prefix))
	if err != nil {
		return "", err
	}
	return decoded, nil
}

func encodeUTF16LEBase64(value string) string {
	encoded := utf16.Encode([]rune(value))
	buf := make([]byte, 0, len(encoded)*2)
	for _, r := range encoded {
		buf = append(buf, byte(r), byte(r>>8))
	}
	return base64.StdEncoding.EncodeToString(buf)
}

func decodeUTF16LEBase64(value string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return "", fmt.Errorf("decode encoded PowerShell command: %w", err)
	}
	if len(data)%2 != 0 {
		return "", fmt.Errorf("encoded PowerShell command has odd byte length")
	}

	u16 := make([]uint16, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		u16 = append(u16, uint16(data[i])|uint16(data[i+1])<<8)
	}
	return string(utf16.Decode(u16)), nil
}

func xmlEscape(value string) string {
	var b bytes.Buffer
	if err := xml.EscapeText(&b, []byte(value)); err != nil {
		return value
	}
	return b.String()
}
