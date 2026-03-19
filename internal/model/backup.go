package model

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meru143/crontui/internal/styles"
)

// loadBackups fetches backups from the backup directory.
func (m *Model) loadBackups() {
	backups, err := modelListBackupsFn(m.cfg)
	if err != nil {
		m.statusMessage = "Error loading backups: " + err.Error()
		m.statusIsError = true
		m.backups = nil
		return
	}
	m.backups = backups
}

// updateBackup handles key events in the backup list view.
func (m Model) updateBackup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "b":
		m.currentView = ViewList
		m.selectedIndex = 0

	case "up", "k":
		if len(m.backups) > 0 {
			m.selectedIndex--
			if m.selectedIndex < 0 {
				m.selectedIndex = len(m.backups) - 1
			}
		}

	case "down", "j":
		if len(m.backups) > 0 {
			m.selectedIndex++
			if m.selectedIndex >= len(m.backups) {
				m.selectedIndex = 0
			}
		}

	case "enter", "r":
		if len(m.backups) > 0 && m.selectedIndex < len(m.backups) {
			backup := m.backups[m.selectedIndex]
			if err := modelRestoreBackupFn(m.cfg, backup.Filename); err != nil {
				m.statusMessage = "Restore failed: " + err.Error()
				m.statusIsError = true
			} else {
				successMessage := fmt.Sprintf("Restored from %s", backup.Filename)
				m.loadJobs()
				if !m.statusIsError {
					m.statusMessage = successMessage
					m.statusIsError = false
				}
			}
			m.currentView = ViewList
			m.selectedIndex = 0
		}

	case "c":
		// Create new backup
		path, err := modelCreateBackupFn(m.cfg)
		if err != nil {
			m.statusMessage = "Backup failed: " + err.Error()
			m.statusIsError = true
		} else {
			m.statusMessage = fmt.Sprintf("Backup created: %s", path)
			m.statusIsError = false
			m.loadBackups()
		}
	}

	return m, nil
}

// viewBackup renders the backup list.
func (m Model) viewBackup() string {
	var b strings.Builder
	w := m.width
	if w == 0 {
		w = 80
	}

	b.WriteString(styles.HeaderStyle.Width(w).Render("🕐 CronTUI — Backups") + "\n\n")

	if len(m.backups) == 0 {
		b.WriteString(styles.HelpStyle.Render("  No backups found. Press 'c' to create one.\n"))
	} else {
		// Header
		headerRow := fmt.Sprintf("  %-*s %-*s %-*s %s",
			30, "Created",
			8, "Jobs",
			10, "Size",
			"Filename",
		)
		b.WriteString(styles.TableHeaderStyle.Render(headerRow) + "\n")
		b.WriteString(styles.HelpStyle.Render(strings.Repeat("─", min(w, 100))) + "\n")

		for i, backup := range m.backups {
			sizeStr := formatBytes(backup.Size)
			row := fmt.Sprintf("  %-*s %-*d %-*s %s",
				30, backup.Created.Format("2006-01-02 15:04:05"),
				8, backup.JobCount,
				10, sizeStr,
				backup.Filename,
			)

			if i == m.selectedIndex {
				b.WriteString(styles.SelectedRowStyle.Width(min(w, 100)).Render(row))
			} else {
				b.WriteString(styles.TableRowStyle.Render(row))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	if m.statusMessage != "" {
		if m.statusIsError {
			b.WriteString(styles.ErrorStyle.Render("  ⚠ "+m.statusMessage) + "\n")
		} else {
			b.WriteString(styles.SuccessStyle.Render("  ✓ "+m.statusMessage) + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(styles.HelpStyle.Render("  ↑/↓ navigate │ ↵/r restore │ c create │ esc back") + "\n")

	return b.String()
}

// formatBytes returns a human-readable file size.
func formatBytes(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	}
}
