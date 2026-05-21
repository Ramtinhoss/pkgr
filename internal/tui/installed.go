package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type InstalledScreen struct {
	svc      *Services
	tbl      table.Model
	pmFilter string
}

func NewInstalledScreen(svc *Services) *InstalledScreen {
	cols := []table.Column{
		{Title: "NAME", Width: 24},
		{Title: "VERSION", Width: 14},
		{Title: "PM", Width: 8},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &InstalledScreen{svc: svc, tbl: t}
}

func NewInstalledScreenFiltered(svc *Services, pmID string) *InstalledScreen {
	cols := []table.Column{
		{Title: "NAME", Width: 24},
		{Title: "VERSION", Width: 14},
		{Title: "PM", Width: 8},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &InstalledScreen{svc: svc, tbl: t, pmFilter: pmID}
}

func (s *InstalledScreen) Name() string { return "installed" }

func (s *InstalledScreen) Init() tea.Cmd {
	return func() tea.Msg {
		pkgs, errs := s.svc.Orc.List(s.svc.Ctx, s.svc.Reg.Active())
		_ = errs
		return loadedPackagesMsg{pkgs: pkgs}
	}
}

type loadedPackagesMsg struct{ pkgs []manager.Package }

func (s *InstalledScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	switch m := msg.(type) {
	case loadedPackagesMsg:
		rows := make([]table.Row, 0, len(m.pkgs))
		for _, p := range m.pkgs {
			if s.pmFilter != "" && p.Manager != s.pmFilter {
				continue
			}
			rows = append(rows, table.Row{p.Name, p.Version, p.Manager})
		}
		s.tbl.SetRows(rows)
	}
	var c tea.Cmd
	s.tbl, c = s.tbl.Update(msg)
	return s, c
}

func (s *InstalledScreen) View() string {
	var b strings.Builder
	title := "Installed"
	if s.pmFilter != "" {
		title = "Installed (" + s.pmFilter + ")"
	}
	b.WriteString(s.svc.Theme.Title.Render(title))
	b.WriteString("\n\n")
	b.WriteString(s.tbl.View())
	return b.String()
}
