package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Screen defines the interface for modular TUI views.
type Screen interface {
	// Name returns the screen identifier.
	Name() string

	// Init returns initial command to execute.
	Init() tea.Cmd

	// Update processes incoming messages and returns updated Screen and Cmd.
	Update(msg tea.Msg) (Screen, tea.Cmd)

	// View returns the rendered view string.
	View() string
}
