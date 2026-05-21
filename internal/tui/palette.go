package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// PaletteScreen provides a command palette for quick command entry.
type PaletteScreen struct {
	name      string
	input     textinput.Model
	services  *Services
}

// NewPaletteScreen creates a new command palette screen.
func NewPaletteScreen(services *Services) *PaletteScreen {
	ti := textinput.New()
	ti.Prompt = ": "
	ti.Placeholder = "search|outdated|pm"
	ti.Focus()
	
	return &PaletteScreen{
		name:     "palette",
		input:    ti,
		services: services,
	}
}

// Name returns the screen name.
func (s *PaletteScreen) Name() string {
	return s.name
}

// Init initializes the screen.
func (s *PaletteScreen) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages for the palette.
func (s *PaletteScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Parse command and dispatch
			cmd := strings.Fields(s.input.Value())
			if len(cmd) == 0 {
				return tea.Cmd(func() tea.Msg {
					return PopScreenMsg{}
				})
			}
			
			switch cmd[0] {
			case "search":
				return tea.Cmd(func() tea.Msg {
					return PushScreenMsg{Screen: NewSearchScreen(s.services)}
				})
			case "outdated":
				return tea.Cmd(func() tea.Msg {
					return PushScreenMsg{Screen: NewOutdatedScreen(s.services)}
				})
			case "pm":
				return tea.Cmd(func() tea.Msg {
					return PushScreenMsg{Screen: NewPMScreen(s.services)}
				})
			}
			
			return tea.Cmd(func() tea.Msg {
				return PopScreenMsg{}
			})
		case tea.KeyEsc:
			return tea.Cmd(func() tea.Msg {
				return PopScreenMsg{}
			})
		}
	}
	
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	return cmd
}

// View renders the command palette.
func (s *PaletteScreen) View() string {
	return fmt.Sprintf("Command Palette\n%s\n", s.input.View())
}
