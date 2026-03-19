package model

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/pkg/types"
)

func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()
	m := New(cfg)

	if m.currentView != ViewList {
		t.Errorf("initial view = %d, want ViewList (%d)", m.currentView, ViewList)
	}
	if m.filter != "all" {
		t.Errorf("initial filter = %q, want %q", m.filter, "all")
	}
}

func TestFilteredJobs_All(t *testing.T) {
	m := Model{
		filter: "all",
		jobs: []types.CronJob{
			{ID: 1, Command: "a", Enabled: true},
			{ID: 2, Command: "b", Enabled: false},
		},
	}
	got := m.filteredJobs()
	if len(got) != 2 {
		t.Errorf("filter=all: got %d, want 2", len(got))
	}
}

func TestFilteredJobs_Enabled(t *testing.T) {
	m := Model{
		filter: "enabled",
		jobs: []types.CronJob{
			{ID: 1, Command: "a", Enabled: true},
			{ID: 2, Command: "b", Enabled: false},
			{ID: 3, Command: "c", Enabled: true},
		},
	}
	got := m.filteredJobs()
	if len(got) != 2 {
		t.Errorf("filter=enabled: got %d, want 2", len(got))
	}
}

func TestFilteredJobs_Disabled(t *testing.T) {
	m := Model{
		filter: "disabled",
		jobs: []types.CronJob{
			{ID: 1, Command: "a", Enabled: true},
			{ID: 2, Command: "b", Enabled: false},
		},
	}
	got := m.filteredJobs()
	if len(got) != 1 {
		t.Errorf("filter=disabled: got %d, want 1", len(got))
	}
	if got[0].ID != 2 {
		t.Errorf("disabled job ID = %d, want 2", got[0].ID)
	}
}

func TestFilteredJobs_Search(t *testing.T) {
	m := Model{
		filter:      "all",
		searchQuery: "backup",
		jobs: []types.CronJob{
			{ID: 1, Command: "/usr/bin/backup", Enabled: true},
			{ID: 2, Command: "/usr/bin/cleanup", Enabled: true},
			{ID: 3, Command: "/usr/bin/other", Description: "Backup helper", Enabled: true},
		},
	}
	got := m.filteredJobs()
	if len(got) != 2 {
		t.Errorf("search=backup: got %d, want 2", len(got))
	}
}

func TestFilteredJobs_SearchCaseInsensitive(t *testing.T) {
	m := Model{
		filter:      "all",
		searchQuery: "BACKUP",
		jobs: []types.CronJob{
			{ID: 1, Command: "/usr/bin/backup", Enabled: true},
			{ID: 2, Command: "/usr/bin/cleanup", Enabled: true},
		},
	}
	got := m.filteredJobs()
	if len(got) != 1 {
		t.Errorf("case-insensitive search: got %d, want 1", len(got))
	}
}

func TestFilteredJobs_SearchAndFilter(t *testing.T) {
	m := Model{
		filter:      "enabled",
		searchQuery: "backup",
		jobs: []types.CronJob{
			{ID: 1, Command: "/usr/bin/backup", Enabled: true},
			{ID: 2, Command: "/usr/bin/backup-old", Enabled: false},
			{ID: 3, Command: "/usr/bin/cleanup", Enabled: true},
		},
	}
	got := m.filteredJobs()
	if len(got) != 1 {
		t.Errorf("search+filter: got %d, want 1", len(got))
	}
}

func TestDemoJobs(t *testing.T) {
	jobs := demoJobs()
	if len(jobs) == 0 {
		t.Fatal("demoJobs() should return non-empty slice")
	}
	for i, j := range jobs {
		if j.ID == 0 {
			t.Errorf("demoJobs[%d].ID should not be 0", i)
		}
		if j.Schedule == "" {
			t.Errorf("demoJobs[%d].Schedule should not be empty", i)
		}
		if j.Command == "" {
			t.Errorf("demoJobs[%d].Command should not be empty", i)
		}
	}
}

func TestViewForm_DoesNotShowWorkingDirField(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewFormAdd

	view := m.viewForm()
	if strings.Contains(view, "Working Dir") {
		t.Fatalf("viewForm should not show working directory field:\n%s", view)
	}
}

func TestViewForm_DoesNotShowMailtoField(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewFormAdd

	view := m.viewForm()
	if strings.Contains(view, "Mailto") {
		t.Fatalf("viewForm should not show mailto field:\n%s", view)
	}
}

func TestLoadJobs_DoesNotUseDemoJobsForReadErrors(t *testing.T) {
	oldReadCrontab := modelReadCrontabFn
	oldGOOS := modelGOOS
	defer func() {
		modelReadCrontabFn = oldReadCrontab
		modelGOOS = oldGOOS
	}()

	modelReadCrontabFn = func() (string, error) {
		return "", errors.New("permission denied")
	}
	modelGOOS = "linux"

	m := New(config.DefaultConfig())
	m.loadJobs()

	if len(m.jobs) != 0 {
		t.Fatalf("expected no demo jobs on non-Windows read error, got %d jobs", len(m.jobs))
	}
	if !m.statusIsError {
		t.Fatal("expected statusIsError to be true")
	}
	if !strings.Contains(m.statusMessage, "permission denied") {
		t.Fatalf("statusMessage = %q, want permission error", m.statusMessage)
	}
}

func TestLoadJobs_UsesDemoJobsOnWindowsOnly(t *testing.T) {
	oldReadCrontab := modelReadCrontabFn
	oldGOOS := modelGOOS
	defer func() {
		modelReadCrontabFn = oldReadCrontab
		modelGOOS = oldGOOS
	}()

	modelReadCrontabFn = func() (string, error) {
		return "", errors.New("crontab unavailable")
	}
	modelGOOS = "windows"

	m := New(config.DefaultConfig())
	m.loadJobs()

	if len(m.jobs) == 0 {
		t.Fatal("expected demo jobs on Windows fallback")
	}
	if !m.statusIsError {
		t.Fatal("expected statusIsError to remain true for Windows fallback")
	}
}

func TestUpdateFormPreview_RebootDescriptorShowsFriendlyMessage(t *testing.T) {
	m := New(config.DefaultConfig())
	m.scheduleInput.SetValue("@reboot")

	m.updateFormPreview()

	if m.formError != "" {
		t.Fatalf("formError = %q, want empty", m.formError)
	}
	if !strings.Contains(strings.ToLower(m.formPreview), "reboot") {
		t.Fatalf("formPreview = %q, want reboot guidance", m.formPreview)
	}
}

func TestUpdateBackup_RestoreKeepsSuccessStatus(t *testing.T) {
	oldReadCrontab := modelReadCrontabFn
	oldRestoreBackup := modelRestoreBackupFn
	defer func() {
		modelReadCrontabFn = oldReadCrontab
		modelRestoreBackupFn = oldRestoreBackup
	}()

	modelReadCrontabFn = func() (string, error) {
		return "0 * * * * /bin/echo restored\n", nil
	}
	modelRestoreBackupFn = func(cfg config.Config, filename string) error {
		return nil
	}

	m := New(config.DefaultConfig())
	m.currentView = ViewBackupList
	m.backups = []types.Backup{{Filename: "restore.bak"}}

	next, _ := m.updateBackup(tea.KeyMsg{Type: tea.KeyEnter})
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("updateBackup returned %T, want model.Model", next)
	}

	if updated.currentView != ViewList {
		t.Fatalf("currentView = %v, want %v", updated.currentView, ViewList)
	}
	if updated.statusIsError {
		t.Fatal("statusIsError = true, want false")
	}
	if updated.statusMessage != "Restored from restore.bak" {
		t.Fatalf("statusMessage = %q, want %q", updated.statusMessage, "Restored from restore.bak")
	}
	if len(updated.jobs) != 1 {
		t.Fatalf("jobs length = %d, want 1", len(updated.jobs))
	}
}

func TestUpdateForm_ScheduleDigitsAreTypedLiterally(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewFormAdd
	m.formFocusIndex = formFieldSchedule
	m.focusCurrentField()

	for _, msg := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'1'}},
		{Type: tea.KeyRunes, Runes: []rune{'-'}},
		{Type: tea.KeyRunes, Runes: []rune{'5'}},
	} {
		next, _ := m.updateForm(msg)
		var ok bool
		m, ok = next.(Model)
		if !ok {
			t.Fatalf("updateForm returned %T, want model.Model", next)
		}
	}

	if got := m.scheduleInput.Value(); got != "1-5" {
		t.Fatalf("scheduleInput.Value() = %q, want %q", got, "1-5")
	}
}

func TestUpdateForm_ScheduleAltDigitAppliesPreset(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewFormAdd
	m.formFocusIndex = formFieldSchedule
	m.focusCurrentField()

	next, _ := m.updateForm(tea.KeyMsg{
		Type:  tea.KeyRunes,
		Runes: []rune{'6'},
		Alt:   true,
	})
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("updateForm returned %T, want model.Model", next)
	}

	if got := updated.scheduleInput.Value(); got != "@reboot" {
		t.Fatalf("scheduleInput.Value() = %q, want %q", got, "@reboot")
	}
}

func TestUpdate_QuestionMarkOpensHelpAndReturnsToPreviousView(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewFormAdd

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want model.Model", next)
	}

	if updated.currentView != ViewHelp {
		t.Fatalf("currentView = %v, want %v", updated.currentView, ViewHelp)
	}
	if updated.previousView != ViewFormAdd {
		t.Fatalf("previousView = %v, want %v", updated.previousView, ViewFormAdd)
	}

	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	restored, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want model.Model", next)
	}
	if restored.currentView != ViewFormAdd {
		t.Fatalf("currentView after leaving help = %v, want %v", restored.currentView, ViewFormAdd)
	}
}

func TestViewForm_ShowsPresetHelp(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewFormAdd

	view := m.viewForm()
	if !strings.Contains(view, "Alt+1..Alt+6") {
		t.Fatalf("viewForm should show preset help:\n%s", view)
	}
}

func TestViewList_EmptyStateMentionsHelp(t *testing.T) {
	m := New(config.DefaultConfig())
	m.currentView = ViewList

	view := m.viewList()
	if !strings.Contains(view, "Press 'a' to add one or '?' for help.") {
		t.Fatalf("viewList should mention help in empty state:\n%s", view)
	}
}

func TestViewHelp_IncludesStableIDGuidance(t *testing.T) {
	m := New(config.DefaultConfig())

	view := m.viewHelp()
	if !strings.Contains(view, "IDs are stable managed IDs") {
		t.Fatalf("viewHelp should mention stable IDs:\n%s", view)
	}
	if !strings.Contains(view, "Alt+6 reboot") {
		t.Fatalf("viewHelp should include preset shortcuts:\n%s", view)
	}
}

func TestUpdateFormPreview_UsesConfiguredShowNextRuns(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ShowNextRuns = 2

	m := New(cfg)
	m.scheduleInput.SetValue("* * * * *")

	m.updateFormPreview()

	lines := strings.Split(strings.TrimSpace(m.formPreview), "\n")
	if len(lines) != 2 {
		t.Fatalf("preview line count = %d, want 2\npreview:\n%s", len(lines), m.formPreview)
	}
}

func TestViewBackup_UsesConfiguredDateFormat(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DateFormat = "2006/01/02 15:04"

	m := New(cfg)
	m.currentView = ViewBackupList
	m.backups = []types.Backup{
		{
			Filename: "backup.bak",
			Created:  time.Date(2026, 3, 19, 14, 30, 0, 0, time.UTC),
			JobCount: 1,
			Size:     128,
		},
	}

	view := m.viewBackup()
	if !strings.Contains(view, "2026/03/19 14:30") {
		t.Fatalf("viewBackup should use configured date format:\n%s", view)
	}
}
