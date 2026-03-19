package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meru143/crontui/internal/styles"
)

// updateRunOutput handles key events in the run output view.
func (m Model) updateRunOutput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q":
		m.currentView = ViewList
	}
	return m, nil
}

// viewRunOutput renders the run output view.
func (m Model) viewRunOutput() string {
	var b strings.Builder
	w := m.width
	if w == 0 {
		w = 80
	}

	b.WriteString(styles.HeaderStyle.Width(w).Render("🕐 CronTUI — Run Output") + "\n\n")
	b.WriteString(styles.PreviewStyle.Render("  Output:") + "\n")

	// Show output lines, indent them
	for _, line := range strings.Split(m.runOutput, "\n") {
		b.WriteString("  " + line + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("  Press Esc, Enter, q, or ? for help") + "\n")

	return b.String()
}

// updateConfirmRemoveAll handles key events in the remove-all confirmation view.
func (m Model) updateConfirmRemoveAll(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if err := modelBackendFn(m.cfg).RemoveAll(m.cfg); err != nil {
			m.statusMessage = "Error removing managed jobs: " + err.Error()
			m.statusIsError = true
		} else {
			m.jobs = nil
			m.selectedIndex = 0
			m.statusMessage = "All managed jobs removed"
			m.statusIsError = false
		}
		m.currentView = ViewList

	case "n", "N", "esc":
		m.currentView = ViewList
	}
	return m, nil
}

// viewConfirmRemoveAll renders the remove-all confirmation dialog.
func (m Model) viewConfirmRemoveAll() string {
	var b strings.Builder
	b.WriteString(m.viewList())
	b.WriteString("\n")
	b.WriteString(styles.ErrorStyle.Render("  ⚠ " + m.confirmMessage))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("  Press y to confirm, n or esc to cancel"))
	b.WriteString("\n")
	return b.String()
}
