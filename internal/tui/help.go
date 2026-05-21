package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// HelpScreen displays keybindings and shortcuts.
type HelpScreen struct {
	name     string
	services *Services
}

// NewHelpScreen creates a new help screen.
func NewHelpScreen(services *Services) *HelpScreen {
	return &HelpScreen{
		name:     "help",
		services: services,
	}
}

// Name returns the screen name.
func (s *HelpScreen) Name() string {
	return s.name
}

// Init initializes the screen.
func (s *HelpScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages for the help screen.
func (s *HelpScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case tea.KeyMsg:
		// Any key closes the help overlay
		return tea.Cmd(func() tea.Msg {
			return PopScreenMsg{}
		})
	case tea.WindowSizeMsg:
		// Handle resizing if needed
		return nil
	}
	return nil
}

// View renders the help screen.
func (s *HelpScreen) View() string {
	title := s.services.Theme.Title.Render("Key Bindings")
	helpText := `
/            - Open command palette
i            - Show installed packages
d            - Mark for delete
u            - Mark for update
o            - Show outdated packages
l            - List installed packages from selected PM
L            - Open operation log
:            - Show help
ESC          - Close overlay
q            - Quit
`
	return fmt.Sprintf("%s\n%s", title, helpText)
}
