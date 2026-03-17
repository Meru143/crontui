package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/meru143/crontui/internal/cli"
	"github.com/meru143/crontui/internal/config"
	"github.com/meru143/crontui/internal/model"
)

func main() {
	// If CLI subcommand provided, run non-interactively
	if cli.Run(os.Args) {
		return
	}

	// Launch interactive TUI
	cfg := config.DefaultConfig()
	m := model.New(cfg)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
