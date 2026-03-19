package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meru143/crontui/internal/cron"
	"github.com/meru143/crontui/internal/crontab"
	"github.com/meru143/crontui/internal/styles"
	"github.com/meru143/crontui/pkg/types"
)

// updateList handles key events in the list view.
func (m Model) updateList(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	jobs := m.filteredJobs()

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "up", "k":
		if len(jobs) > 0 {
			m.selectedIndex--
			if m.selectedIndex < 0 {
				m.selectedIndex = len(jobs) - 1
			}
		}

	case "down", "j":
		if len(jobs) > 0 {
			m.selectedIndex++
			if m.selectedIndex >= len(jobs) {
				m.selectedIndex = 0
			}
		}

	case "home", "g":
		m.selectedIndex = 0

	case "end", "G":
		if len(jobs) > 0 {
			m.selectedIndex = len(jobs) - 1
		}

	case "a":
		m.currentView = ViewFormAdd
		m.editingJob = nil
		m.scheduleInput.SetValue("")
		m.commandInput.SetValue("")
		m.descriptionInput.SetValue("")
		m.workingDirInput.SetValue("")
		m.mailtoInput.SetValue("")
		m.scheduleInput.Focus()
		m.formFocusIndex = 0
		m.formError = ""
		m.formPreview = ""

	case "e", "enter":
		if len(jobs) > 0 && m.selectedIndex < len(jobs) {
			job := jobs[m.selectedIndex]
			m.currentView = ViewFormEdit
			m.editingJob = &job
			m.scheduleInput.SetValue(job.Schedule)
			m.commandInput.SetValue(job.Command)
			m.descriptionInput.SetValue(job.Description)
			m.workingDirInput.SetValue(job.WorkingDir)
			m.mailtoInput.SetValue(job.Mailto)
			m.scheduleInput.Focus()
			m.formFocusIndex = 0
			m.formError = ""
			m.updateFormPreview()
		}

	case "d":
		if len(jobs) > 0 && m.selectedIndex < len(jobs) {
			job := jobs[m.selectedIndex]
			m.currentView = ViewConfirmDelete
			m.confirmMessage = fmt.Sprintf("Delete job: %s ?", truncate(job.Command, 40))
		}

	case "t":
		// Toggle enabled/disabled
		if len(jobs) > 0 && m.selectedIndex < len(jobs) {
			job := jobs[m.selectedIndex]
			for i := range m.jobs {
				if m.jobs[i].ID == job.ID {
					m.jobs[i].Enabled = !m.jobs[i].Enabled
					break
				}
			}
			if err := crontab.WriteJobsWithBackup(m.cfg, m.jobs); err != nil {
				m.statusMessage = "Error: " + err.Error()
				m.statusIsError = true
			} else {
				m.statusMessage = "Job toggled successfully"
				m.statusIsError = false
			}
		}

	case "/":
		m.currentView = ViewSearch
		m.searchMode = true
		m.searchQuery = ""

	case "f":
		// Cycle filter: all → enabled → disabled → all
		switch m.filter {
		case "all":
			m.filter = "enabled"
		case "enabled":
			m.filter = "disabled"
		default:
			m.filter = "all"
		}
		m.selectedIndex = 0

	case "b":
		m.currentView = ViewBackupList
		m.loadBackups()
		m.selectedIndex = 0

	case "r":
		m.loadJobs()
		m.statusMessage = "Refreshed"
		m.statusIsError = false

	case "x":
		// Run Now
		if len(jobs) > 0 && m.selectedIndex < len(jobs) {
			job := jobs[m.selectedIndex]
			if !job.Enabled {
				m.statusMessage = "Cannot run disabled job"
				m.statusIsError = true
			} else {
				out, err := crontab.ExecCommand(job.Command).CombinedOutput()
				if err != nil {
					m.runOutput = fmt.Sprintf("Error: %s\n\n%s", err, string(out))
				} else if len(out) == 0 {
					m.runOutput = "(no output)"
				} else {
					m.runOutput = string(out)
				}
				m.currentView = ViewRunOutput
			}
		}

	case "R":
		// Remove all crontab
		m.currentView = ViewConfirmRemoveAll
		m.confirmMessage = "Remove ALL crontab entries? This cannot be undone!"
	}

	return m, nil
}

// viewList renders the job list view.
func (m Model) viewList() string {
	var b strings.Builder
	w := m.width
	if w == 0 {
		w = 80
	}

	// Header
	header := styles.HeaderStyle.Width(w).Render("🕐 CronTUI — Cron Job Manager")
	b.WriteString(header + "\n\n")

	jobs := m.filteredJobs()

	// Filter indicator
	filterText := ""
	switch m.filter {
	case "enabled":
		filterText = styles.SuccessStyle.Render(" [Filter: Enabled] ")
	case "disabled":
		filterText = styles.ErrorStyle.Render(" [Filter: Disabled] ")
	}

	if m.searchQuery != "" {
		filterText += styles.HelpStyle.Render(fmt.Sprintf(" Search: \"%s\" ", m.searchQuery))
	}

	if filterText != "" {
		b.WriteString(filterText + "\n\n")
	}

	if len(jobs) == 0 {
		b.WriteString(styles.HelpStyle.Render("  No cron jobs found. Press 'a' to add one.\n"))
	} else {
		// Table header
		colID := 4
		colStatus := 9
		colSchedule := 17
		colNextRun := 22
		colCmd := w - colID - colStatus - colSchedule - colNextRun - 10
		if colCmd < 20 {
			colCmd = 20
		}

		headerRow := fmt.Sprintf("  %-*s %-*s %-*s %-*s %s",
			colID, "#",
			colStatus, "Status",
			colSchedule, "Schedule",
			colNextRun, "Next Run",
			"Command",
		)
		b.WriteString(styles.TableHeaderStyle.Render(headerRow) + "\n")
		b.WriteString(styles.HelpStyle.Render(strings.Repeat("─", min(w, 120))) + "\n")

		// Rows
		for i, job := range jobs {
			status := styles.EnabledStyle.Render("● ON ")
			if !job.Enabled {
				status = styles.DisabledStyle.Render("○ OFF")
			}

			nextRun := ""
			if job.Enabled {
				runs, err := cron.NextRuns(job.Schedule, 1)
				if err == nil && len(runs) > 0 {
					nextRun = cron.HumanReadable(runs[0])
				}
			}

			row := fmt.Sprintf("  %-*d %s %-*s %-*s %s",
				colID, job.ID,
				status,
				colSchedule, job.Schedule,
				colNextRun, truncate(nextRun, colNextRun),
				truncate(job.Command, colCmd),
			)

			if i == m.selectedIndex {
				b.WriteString(styles.SelectedRowStyle.Width(min(w, 120)).Render(row))
			} else {
				b.WriteString(styles.TableRowStyle.Render(row))
			}
			b.WriteString("\n")

			// Show description if present
			if job.Description != "" && i == m.selectedIndex {
				desc := styles.HelpStyle.Render(fmt.Sprintf("    📝 %s", job.Description))
				b.WriteString(desc + "\n")
			}
		}
	}

	b.WriteString("\n")

	// Status bar
	if m.statusMessage != "" {
		if m.statusIsError {
			b.WriteString(styles.ErrorStyle.Render("  ⚠ "+m.statusMessage) + "\n")
		} else {
			b.WriteString(styles.SuccessStyle.Render("  ✓ "+m.statusMessage) + "\n")
		}
		b.WriteString("\n")
	}

	// Help
	help := []string{
		"↑/↓ navigate", "a add", "e/↵ edit", "d delete", "t toggle",
		"x run now", "/ search", "f filter", "b backups", "R remove all", "r refresh", "q quit",
	}
	b.WriteString(styles.HelpStyle.Render("  " + strings.Join(help, " │ ")))
	b.WriteString("\n")

	return b.String()
}

// updateSearch handles key events in search mode.
func (m Model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.currentView = ViewList
		m.searchMode = false
		m.selectedIndex = 0
	case "esc":
		m.currentView = ViewList
		m.searchMode = false
		m.searchQuery = ""
		m.selectedIndex = 0
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
		}
	}
	return m, nil
}

// updateConfirmDelete handles the delete confirmation dialog.
func (m Model) updateConfirmDelete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		jobs := m.filteredJobs()
		if m.selectedIndex < len(jobs) {
			job := jobs[m.selectedIndex]
			// Remove from m.jobs
			newJobs := make([]types.CronJob, 0, len(m.jobs)-1)
			for _, j := range m.jobs {
				if j.ID != job.ID {
					newJobs = append(newJobs, j)
				}
			}
			m.jobs = newJobs

			if err := crontab.WriteJobsWithBackup(m.cfg, m.jobs); err != nil {
				m.statusMessage = "Error deleting: " + err.Error()
				m.statusIsError = true
			} else {
				m.statusMessage = "Job deleted"
				m.statusIsError = false
			}

			if m.selectedIndex >= len(m.filteredJobs()) {
				m.selectedIndex = max(0, len(m.filteredJobs())-1)
			}
		}
		m.currentView = ViewList

	case "n", "N", "esc":
		m.currentView = ViewList
	}
	return m, nil
}

// viewConfirmDelete renders the delete confirmation dialog.
func (m Model) viewConfirmDelete() string {
	var b strings.Builder
	b.WriteString(m.viewList())
	b.WriteString("\n")
	b.WriteString(styles.ErrorStyle.Render("  ⚠ " + m.confirmMessage))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("  Press y to confirm, n or esc to cancel"))
	b.WriteString("\n")
	return b.String()
}
