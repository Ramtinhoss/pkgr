package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// opEntry represents a single operation in the log.
type opEntry struct {
	When time.Time
	Text string
}

// OpLogScreen displays a scrollable history of package operations.
type OpLogScreen struct {
	name     string
	viewport viewport.Model
	entries  []opEntry
	services *Services
}

// NewOpLogScreen creates a new operation log screen.
func NewOpLogScreen(services *Services) *OpLogScreen {
	return &OpLogScreen{
		name:     "oplog",
		viewport: viewport.New(80, 20),
		entries:  []opEntry{},
		services: services,
	}
}

// Name returns the screen name.
func (s *OpLogScreen) Name() string {
	return s.name
}

// Init initializes the screen.
func (s *OpLogScreen) Init() tea.Cmd {
	return nil
}

// Update handles messages for the operation log.
func (s *OpLogScreen) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case OpStartMsg:
		s.entries = append(s.entries, opEntry{
			When: time.Now(),
			Text: string(msg.Op),
		})
		s.refresh()
		return nil
	case OpDoneMsg:
		s.entries = append(s.entries, opEntry{
			When: time.Now(),
			Text: fmt.Sprintf("%s (done)", msg.Op),
		})
		s.refresh()
		return nil
	case tea.WindowSizeMsg:
		s.viewport.Width = msg.Width
		s.viewport.Height = msg.Height - 3
		s.refresh()
		return nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			s.viewport.LineUp(1)
		case "down":
			s.viewport.LineDown(1)
		case "pgup":
			s.viewport.PageUp()
		case "pgdn":
			s.viewport.PageDown()
		}
		return nil
	}
	return nil
}

// View renders the operation log.
func (s *OpLogScreen) View() string {
	title := s.services.Theme.Subtle.Render("Operation Log")
	content := s.viewport.View()
	return fmt.Sprintf("%s\n%s", title, content)
}

// refresh updates the viewport with current entries.
func (s *OpLogScreen) refresh() {
	content := ""
	for _, entry := range s.entries {
		content += fmt.Sprintf("%s | %s\n", entry.When.Format("15:04:05"), entry.Text)
	}
	s.viewport.SetContent(content)
}
