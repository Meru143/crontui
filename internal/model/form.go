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

const (
	formFieldSchedule    = 0
	formFieldCommand     = 1
	formFieldDescription = 2
	formFieldCount       = 3
)

// updateForm handles key events in the add/edit form view.
func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1", "2", "3", "4", "5", "6":
		if m.formFocusIndex == formFieldSchedule {
			presets := map[string]string{
				"1": "0 * * * *",
				"2": "0 0 * * *",
				"3": "0 0 * * 0",
				"4": "0 0 1 * *",
				"5": "0 0 1 1 *",
				"6": "@reboot",
			}
			m.scheduleInput.SetValue(presets[msg.String()])
			m.scheduleInput.CursorEnd()
			m.updateFormPreview()
			return m, nil
		}
		// Not on schedule field — forward to active input
		var cmd tea.Cmd
		switch m.formFocusIndex {
		case formFieldCommand:
			m.commandInput, cmd = m.commandInput.Update(msg)
		case formFieldDescription:
			m.descriptionInput, cmd = m.descriptionInput.Update(msg)
		}
		return m, cmd

	case "esc":
		m.currentView = ViewList
		m.formError = ""
		return m, nil

	case "tab", "shift+tab":
		if msg.String() == "tab" {
			m.formFocusIndex = (m.formFocusIndex + 1) % formFieldCount
		} else {
			m.formFocusIndex = (m.formFocusIndex - 1 + formFieldCount) % formFieldCount
		}
		m.focusCurrentField()
		return m, nil

	case "ctrl+s":
		return m.saveForm()

	default:
		// Update the focused input
		var cmd tea.Cmd
		switch m.formFocusIndex {
		case formFieldSchedule:
			m.scheduleInput, cmd = m.scheduleInput.Update(msg)
			m.updateFormPreview()
		case formFieldCommand:
			m.commandInput, cmd = m.commandInput.Update(msg)
		case formFieldDescription:
			m.descriptionInput, cmd = m.descriptionInput.Update(msg)
		}
		return m, cmd
	}
}

// focusCurrentField focuses the correct text input.
func (m *Model) focusCurrentField() {
	m.scheduleInput.Blur()
	m.commandInput.Blur()
	m.descriptionInput.Blur()
	m.workingDirInput.Blur()
	m.mailtoInput.Blur()

	switch m.formFocusIndex {
	case formFieldSchedule:
		m.scheduleInput.Focus()
	case formFieldCommand:
		m.commandInput.Focus()
	case formFieldDescription:
		m.descriptionInput.Focus()
	}
}

// updateFormPreview updates the next-run preview based on current schedule input.
func (m *Model) updateFormPreview() {
	schedule := m.scheduleInput.Value()
	if schedule == "" {
		m.formPreview = ""
		m.formError = ""
		return
	}

	valid, runs, hr, errMsg := cron.Preview(schedule, 5)
	if !valid {
		m.formError = errMsg
		m.formPreview = ""
		return
	}

	m.formError = ""
	var lines []string
	for i, t := range runs {
		lines = append(lines, fmt.Sprintf("  %d. %s  (%s)", i+1, t.Format("Mon Jan 2 15:04:05"), hr[i]))
	}

	m.formPreview = strings.Join(lines, "\n")
}

// saveForm validates and saves the form data.
func (m Model) saveForm() (tea.Model, tea.Cmd) {
	schedule := strings.TrimSpace(m.scheduleInput.Value())
	command := strings.TrimSpace(m.commandInput.Value())
	description := strings.TrimSpace(m.descriptionInput.Value())

	// Validate
	if schedule == "" {
		m.formError = "Schedule is required"
		return m, nil
	}
	if command == "" {
		m.formError = "Command is required"
		return m, nil
	}

	valid, err := cron.Validate(schedule)
	if !valid || err != nil {
		m.formError = "Invalid schedule: " + err.Error()
		return m, nil
	}

	if m.currentView == ViewFormAdd {
		// Find next ID
		maxID := 0
		for _, j := range m.jobs {
			if j.ID > maxID {
				maxID = j.ID
			}
		}
		m.jobs = append(m.jobs, types.CronJob{
			ID:          maxID + 1,
			Schedule:    schedule,
			Command:     command,
			Description: description,
			Enabled:     true,
		})
		m.statusMessage = "Job added successfully"
	} else {
		// Edit existing
		for i := range m.jobs {
			if m.jobs[i].ID == m.editingJob.ID {
				m.jobs[i].Schedule = schedule
				m.jobs[i].Command = command
				m.jobs[i].Description = description
				break
			}
		}
		m.statusMessage = "Job updated successfully"
	}

	m.statusIsError = false

	// Write to crontab
	if err := crontab.WriteJobsWithBackup(m.cfg, m.jobs); err != nil {
		m.statusMessage = "Saved in memory but failed to write crontab: " + err.Error()
		m.statusIsError = true
	}

	m.currentView = ViewList
	m.formError = ""
	return m, nil
}

// viewForm renders the add/edit form.
func (m Model) viewForm() string {
	var b strings.Builder
	w := m.width
	if w == 0 {
		w = 80
	}

	title := "Add New Cron Job"
	if m.currentView == ViewFormEdit {
		title = "Edit Cron Job"
	}

	b.WriteString(styles.HeaderStyle.Width(w).Render("🕐 CronTUI — "+title) + "\n\n")

	// Schedule field
	scheduleLabel := "  Schedule:"
	if m.formFocusIndex == formFieldSchedule {
		scheduleLabel = styles.TitleStyle.Render(scheduleLabel)
	}
	b.WriteString(scheduleLabel + "\n")
	b.WriteString("  " + m.scheduleInput.View() + "\n")

	// Quick presets
	presets := styles.HelpStyle.Render("  Presets: */5→every 5min  0 *→hourly  0 0 *→daily  @reboot  @weekly")
	b.WriteString(presets + "\n\n")

	// Validation + preview
	if m.formError != "" {
		b.WriteString(styles.ErrorStyle.Render("  ✗ "+m.formError) + "\n\n")
	} else if m.formPreview != "" {
		b.WriteString(styles.SuccessStyle.Render("  ✓ Valid schedule") + "\n")
		b.WriteString(styles.PreviewStyle.Render("  Next runs:") + "\n")
		b.WriteString(m.formPreview + "\n\n")
	} else {
		b.WriteString("\n")
	}

	// Command field
	commandLabel := "  Command:"
	if m.formFocusIndex == formFieldCommand {
		commandLabel = styles.TitleStyle.Render(commandLabel)
	}
	b.WriteString(commandLabel + "\n")
	b.WriteString("  " + m.commandInput.View() + "\n\n")

	// Description field
	descLabel := "  Description (optional):"
	if m.formFocusIndex == formFieldDescription {
		descLabel = styles.TitleStyle.Render(descLabel)
	}
	b.WriteString(descLabel + "\n")
	b.WriteString("  " + m.descriptionInput.View() + "\n\n")

	// Help
	b.WriteString(styles.HelpStyle.Render("  Tab cycle fields │ Ctrl+S save │ Esc cancel") + "\n")

	return b.String()
}
