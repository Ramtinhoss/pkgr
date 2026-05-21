package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type OutdatedScreen struct {
	svc *Services
	tbl table.Model
}

func NewOutdatedScreen(svc *Services) *OutdatedScreen {
	cols := []table.Column{
		{Title: "NAME", Width: 24},
		{Title: "CURRENT", Width: 14},
		{Title: "LATEST", Width: 14},
		{Title: "PM", Width: 8},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &OutdatedScreen{svc: svc, tbl: t}
}

func (s *OutdatedScreen) Name() string { return "outdated" }

func (s *OutdatedScreen) Init() tea.Cmd {
	return func() tea.Msg {
		pkgs, _ := s.svc.Orc.Outdated(s.svc.Ctx, s.svc.Reg.Active())
		return loadedOutdatedMsg{pkgs: pkgs}
	}
}

type loadedOutdatedMsg struct{ pkgs []manager.Package }

func (s *OutdatedScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if m, ok := msg.(loadedOutdatedMsg); ok {
		rows := make([]table.Row, 0, len(m.pkgs))
		for _, p := range m.pkgs {
			rows = append(rows, table.Row{p.Name, p.Version, p.Latest, p.Manager})
		}
		s.tbl.SetRows(rows)
	}
	var c tea.Cmd
	s.tbl, c = s.tbl.Update(msg)
	return s, c
}

func (s *OutdatedScreen) View() string {
	var b strings.Builder
	b.WriteString(s.svc.Theme.Title.Render("Outdated"))
	b.WriteString("\n\n")
	b.WriteString(s.tbl.View())
	return b.String()
}
