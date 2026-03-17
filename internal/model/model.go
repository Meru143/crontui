package model

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/internal/crontab"
	"github.com/meru143/crontui/pkg/types"
)

// ViewType represents the current screen in the TUI.
type ViewType int

const (
	ViewList ViewType = iota
	ViewFormAdd
	ViewFormEdit
	ViewBackupList
	ViewJobDetail
	ViewConfirmDelete
	ViewSearch
	ViewRunOutput
	ViewConfirmRemoveAll
)

// Model is the top-level Bubble Tea model.
type Model struct {
	// Data
	jobs    []types.CronJob
	backups []types.Backup
	cfg     config.Config

	// View state
	currentView   ViewType
	selectedIndex int
	editingJob    *types.CronJob

	// List state
	searchQuery string
	searchMode  bool

	filter      string // "all", "enabled", "disabled"

	// Form inputs
	scheduleInput    textinput.Model
	commandInput     textinput.Model
	descriptionInput textinput.Model
	workingDirInput  textinput.Model
	mailtoInput      textinput.Model
	formFocusIndex   int
	formError        string
	formPreview      string

	// Run output
	runOutput string

	// Status
	statusMessage string
	statusIsError bool

	// Terminal size
	width  int
	height int

	// Confirm dialog
	confirmMessage string
	confirmAction  func() tea.Msg
}

// reloadMsg signals that we should reload jobs from crontab.
type reloadMsg struct{}

// statusMsg sets the status bar message.
type statusMsg struct {
	message string
	isError bool
}

// New creates and initializes the Model.
func New(cfg config.Config) Model {
	si := textinput.New()
	si.Placeholder = "*/5 * * * *"
	si.CharLimit = 64

	ci := textinput.New()
	ci.Placeholder = "/path/to/command --flag"
	ci.CharLimit = 256

	di := textinput.New()
	di.Placeholder = "Description (optional)"
	di.CharLimit = 128

	wdi := textinput.New()
	wdi.Placeholder = "/home/user (optional)"
	wdi.CharLimit = 256

	mi := textinput.New()
	mi.Placeholder = "user@example.com (optional)"
	mi.CharLimit = 128

	return Model{
		cfg:              cfg,
		currentView:      ViewList,
		filter:           "all",

		scheduleInput:    si,
		commandInput:     ci,
		descriptionInput: di,
		workingDirInput:  wdi,
		mailtoInput:      mi,
	}
}

// Init loads the initial jobs from crontab.
func (m Model) Init() tea.Cmd {
	return func() tea.Msg { return reloadMsg{} }
}

// Update handles all messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case reloadMsg:
		m.loadJobs()
		return m, nil

	case statusMsg:
		m.statusMessage = msg.message
		m.statusIsError = msg.isError
		return m, nil

	case tea.KeyMsg:
		// Global quit
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Route by view
		switch m.currentView {
		case ViewList:
			return m.updateList(msg)
		case ViewFormAdd, ViewFormEdit:
			return m.updateForm(msg)
		case ViewBackupList:
			return m.updateBackup(msg)
		case ViewConfirmDelete:
			return m.updateConfirmDelete(msg)
		case ViewConfirmRemoveAll:
			return m.updateConfirmRemoveAll(msg)
		case ViewSearch:
			return m.updateSearch(msg)
		case ViewRunOutput:
			return m.updateRunOutput(msg)
		}
	}

	return m, nil
}

// View renders the current screen.
func (m Model) View() string {
	switch m.currentView {
	case ViewFormAdd, ViewFormEdit:
		return m.viewForm()
	case ViewBackupList:
		return m.viewBackup()
	case ViewConfirmDelete:
		return m.viewConfirmDelete()
	case ViewConfirmRemoveAll:
		return m.viewConfirmRemoveAll()
	case ViewRunOutput:
		return m.viewRunOutput()
	default:
		return m.viewList()
	}
}

// loadJobs reads jobs from crontab.
func (m *Model) loadJobs() {
	raw, err := crontab.ReadCrontab()
	if err != nil {
		m.statusMessage = "Error reading crontab: " + err.Error()
		m.statusIsError = true
		// Use demo jobs on Windows or when crontab fails
		m.jobs = demoJobs()
		return
	}

	if raw == "" {
		m.jobs = nil
		m.statusMessage = "No crontab entries found"
		m.statusIsError = false
		return
	}

	m.jobs = crontab.ParseCrontab(raw)
	m.statusMessage = ""
}

// filteredJobs returns jobs matching the current filter and search.
func (m Model) filteredJobs() []types.CronJob {
	var result []types.CronJob
	for _, j := range m.jobs {
		// Apply filter
		switch m.filter {
		case "enabled":
			if !j.Enabled {
				continue
			}
		case "disabled":
			if j.Enabled {
				continue
			}
		}

		// Apply search
		if m.searchQuery != "" {
			if !containsIgnoreCase(j.Command, m.searchQuery) &&
				!containsIgnoreCase(j.Schedule, m.searchQuery) &&
				!containsIgnoreCase(j.Description, m.searchQuery) {
				continue
			}
		}

		result = append(result, j)
	}
	return result
}

// demoJobs returns sample jobs for display when crontab is unavailable.
func demoJobs() []types.CronJob {
	return []types.CronJob{
		{ID: 1, Schedule: "*/5 * * * *", Command: "/usr/local/bin/healthcheck", Description: "Health check every 5 min", Enabled: true},
		{ID: 2, Schedule: "0 2 * * *", Command: "/opt/backup/run-backup.sh", Description: "Nightly database backup", Enabled: true},
		{ID: 3, Schedule: "0 * * * *", Command: "/usr/bin/logrotate /etc/logrotate.conf", Description: "Rotate logs hourly", Enabled: true},
		{ID: 4, Schedule: "30 6 * * 1-5", Command: "/home/user/scripts/morning-report.sh", Description: "Weekday morning report", Enabled: false},
		{ID: 5, Schedule: "0 0 1 * *", Command: "/opt/cleanup/prune-old-data.sh --days 90", Description: "Monthly data cleanup", Enabled: true},
	}
}
