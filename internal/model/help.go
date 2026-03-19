package model

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meru143/crontui/internal/styles"
)

// updateHelp handles key events in the help view.
func (m Model) updateHelp(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter", "q", "?":
		m.currentView = m.previousView
		return m, nil
	default:
		return m, nil
	}
}

// viewHelp renders the help screen.
func (m Model) viewHelp() string {
	var b strings.Builder
	w := m.width
	if w == 0 {
		w = 80
	}

	b.WriteString(styles.HeaderStyle.Width(w).Render("🕐 CronTUI — Help") + "\n\n")

	sections := []string{
		"  List View",
		"    ↑/↓ or j/k navigate",
		"    a add  e edit  d delete  t toggle  x run now",
		"    / search  f filter  b backups  R remove all  r refresh",
		"",
		"  Form View",
		"    Tab cycles fields  Ctrl+S saves  Esc cancels",
		"    Alt+1 hourly  Alt+2 daily  Alt+3 weekly",
		"    Alt+4 monthly  Alt+5 yearly  Alt+6 reboot",
		"",
		"  Backups",
		"    c create backup  Enter or r restore selected backup",
		"",
		"  Notes",
		"    IDs are stable managed IDs and do not renumber after deletes.",
		"    Native Windows manages tasks only inside the configured Task Scheduler path.",
		"    Press ? again, Esc, Enter, or q to return.",
	}

	for _, line := range sections {
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.TrimSpace(line) != "" {
			b.WriteString(styles.TitleStyle.Render(line) + "\n")
			continue
		}
		b.WriteString(line + "\n")
	}

	return b.String()
}
