package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type HomeScreen struct {
	svc *Services
}

func NewHomeScreen(svc *Services) *HomeScreen { return &HomeScreen{svc: svc} }

func (h *HomeScreen) Name() string  { return "home" }
func (h *HomeScreen) Init() tea.Cmd { return nil }

func (h *HomeScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if m, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(m, h.svc.Keys.Search):
			return h, func() tea.Msg { return PushScreenMsg{Screen: NewSearchScreen(h.svc)} }
		case key.Matches(m, h.svc.Keys.Outdated):
			return h, func() tea.Msg { return PushScreenMsg{Screen: NewOutdatedScreen(h.svc)} }
		}
	}
	return h, nil
}

func (h *HomeScreen) View() string {
	var b strings.Builder
	b.WriteString(h.svc.Theme.Title.Render("Detected package managers"))
	b.WriteString("\n\n")
	for _, m := range h.svc.Reg.Active() {
		fmt.Fprintf(&b, "  %s %s\n", h.svc.Theme.Accent.Render("●"), m.ID())
	}
	b.WriteString("\n")
	b.WriteString(h.svc.Theme.Subtle.Render("Press / to search, o for outdated, ? for help"))
	return b.String()
}

// TODO: replace with real impls in T4/T6
func NewSearchScreen(svc *Services) Screen   { return &stubScreen{name: "search"} }
func NewOutdatedScreen(svc *Services) Screen { return &stubScreen{name: "outdated"} }

type stubScreen struct{ name string }

func (s *stubScreen) Name() string                                { return s.name }
func (s *stubScreen) Init() tea.Cmd                               { return nil }
func (s *stubScreen) Update(msg tea.Msg) (Screen, tea.Cmd)        { return s, nil }
func (s *stubScreen) View() string                                { return s.name + " (stub)" }
