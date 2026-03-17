package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#7C3AED") // Purple
	SecondaryColor = lipgloss.Color("#06B6D4") // Cyan
	SuccessColor   = lipgloss.Color("#10B981") // Green
	ErrorColor     = lipgloss.Color("#EF4444") // Red
	WarningColor   = lipgloss.Color("#F59E0B") // Amber
	MutedColor     = lipgloss.Color("#6B7280") // Gray
	TextColor      = lipgloss.Color("#E5E7EB") // Light gray
	BgColor        = lipgloss.Color("#1F2937") // Dark background

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor).
			BorderStyle(lipgloss.DoubleBorder()).
			BorderBottom(true).
			BorderForeground(PrimaryColor).
			Padding(0, 1)

	// Title
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(PrimaryColor)

	// Table header
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(SecondaryColor).
				Padding(0, 1)

	// Table row (normal)
	TableRowStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Padding(0, 1)

	// Table row (selected)
	SelectedRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(PrimaryColor).
				Bold(true).
				Padding(0, 1)

	// Status: enabled
	EnabledStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	// Status: disabled
	DisabledStyle = lipgloss.NewStyle().
			Foreground(ErrorColor)

	// Input field
	InputStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(0, 1)

	// Input field (focused)
	FocusedInputStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(SecondaryColor).
				Padding(0, 1)

	// Error message
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	// Success message
	SuccessStyle = lipgloss.NewStyle().
			Foreground(SuccessColor).
			Bold(true)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(MutedColor)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(TextColor).
			Background(lipgloss.Color("#374151")).
			Padding(0, 1)

	// Preview panel
	PreviewStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(SecondaryColor).
			Padding(0, 1)
)
