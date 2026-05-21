// Package tui hosts the bubbletea TUI for pkgr.
package tui

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Title       lipgloss.Style
	Subtle      lipgloss.Style
	Accent      lipgloss.Style
	Warning     lipgloss.Style
	Error       lipgloss.Style
	StatusBar   lipgloss.Style
	HelpBar     lipgloss.Style
	TableHead   lipgloss.Style
	TableRow    lipgloss.Style
	TableRowAlt lipgloss.Style
	Selected    lipgloss.Style
}

func DefaultTheme(dark bool) Theme {
	bg := lipgloss.Color("16")
	fg := lipgloss.Color("231")
	accent := lipgloss.Color("39")
	subtle := lipgloss.Color("241")
	if !dark {
		bg = lipgloss.Color("231")
		fg = lipgloss.Color("16")
		accent = lipgloss.Color("33")
		subtle = lipgloss.Color("245")
	}
	_ = bg
	return Theme{
		Title:       lipgloss.NewStyle().Foreground(accent).Bold(true),
		Subtle:      lipgloss.NewStyle().Foreground(subtle),
		Accent:      lipgloss.NewStyle().Foreground(accent),
		Warning:     lipgloss.NewStyle().Foreground(lipgloss.Color("214")),
		Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		StatusBar:   lipgloss.NewStyle().Foreground(fg).Background(lipgloss.Color("236")).Padding(0, 1),
		HelpBar:     lipgloss.NewStyle().Foreground(subtle),
		TableHead:   lipgloss.NewStyle().Foreground(accent).Bold(true).Underline(true),
		TableRow:    lipgloss.NewStyle(),
		TableRowAlt: lipgloss.NewStyle().Foreground(subtle),
		Selected:    lipgloss.NewStyle().Background(accent).Foreground(lipgloss.Color("16")).Bold(true),
	}
}
