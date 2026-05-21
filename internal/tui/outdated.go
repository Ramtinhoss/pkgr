package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

// column indices for outdated table (marker prepended)
const (
	oColMarker = 0
	oColName   = 1
	oColCur    = 2
	oColLatest = 3
	oColPM     = 4
)

type OutdatedScreen struct {
	svc  *Services
	tbl  table.Model
	pkgs []manager.Package // backing slice
	sel  Selection
}

func NewOutdatedScreen(svc *Services) *OutdatedScreen {
	cols := []table.Column{
		{Title: " ", Width: 2},
		{Title: "NAME", Width: 24},
		{Title: "CURRENT", Width: 14},
		{Title: "LATEST", Width: 14},
		{Title: "PM", Width: 8},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &OutdatedScreen{svc: svc, tbl: t, sel: NewSelection()}
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
	switch m := msg.(type) {
	case loadedOutdatedMsg:
		s.pkgs = m.pkgs
		s.refreshTable()

	case tea.KeyMsg:
		row := s.tbl.SelectedRow()
		pkg := s.pkgFromRow(row)

		switch {
		case key.Matches(m, s.svc.Keys.ToggleSelect) && pkg.Name != "":
			s.sel.Toggle(pkg)
			s.refreshTable()
			return s, nil

		case key.Matches(m, s.svc.Keys.SelectAll):
			for _, p := range s.pkgs {
				s.sel[KeyFor(p)] = p
			}
			s.refreshTable()
			return s, nil

		case key.Matches(m, s.svc.Keys.ClearSelect):
			s.sel.Clear()
			s.refreshTable()
			return s, nil

		case key.Matches(m, s.svc.Keys.Update):
			if s.sel.Count() > 0 {
				return s, requestUpdateMany(s.svc, s.sel)
			}
			if pkg.Name != "" {
				return s, requestUpdate(s.svc, pkg.Manager, pkg.Name)
			}

		case m.String() == "U":
			// Update everything visible regardless of selection
			if len(s.pkgs) > 0 {
				allSel := NewSelection()
				for _, p := range s.pkgs {
					allSel[KeyFor(p)] = p
				}
				return s, requestUpdateMany(s.svc, allSel)
			}
		}
	}

	var c tea.Cmd
	s.tbl, c = s.tbl.Update(msg)
	return s, c
}

func (s *OutdatedScreen) pkgFromRow(row []string) manager.Package {
	if len(row) <= oColPM {
		return manager.Package{}
	}
	name := row[oColName]
	pm := row[oColPM]
	for _, p := range s.pkgs {
		if p.Name == name && p.Manager == pm {
			return p
		}
	}
	return manager.Package{Name: name, Manager: pm}
}

func (s *OutdatedScreen) refreshTable() {
	rows := make([]table.Row, 0, len(s.pkgs))
	for _, p := range s.pkgs {
		marker := " "
		if s.sel.Has(p) {
			marker = "✓"
		}
		rows = append(rows, table.Row{marker, p.Name, p.Version, p.Latest, p.Manager})
	}
	s.tbl.SetRows(rows)
}

func (s *OutdatedScreen) View() string {
	var b strings.Builder
	b.WriteString(s.svc.Theme.Title.Render("Outdated"))
	if s.sel.Count() > 0 {
		b.WriteString(fmt.Sprintf("  %s", s.svc.Theme.Warning.Render(fmt.Sprintf("%d selected", s.sel.Count()))))
	}
	b.WriteString("\n\n")
	b.WriteString(s.tbl.View())
	b.WriteString("\n")
	b.WriteString(s.svc.Theme.Subtle.Render("[u] update selected  [U] update all visible  [space] toggle  [a] sel-all  [A] clear  [esc] back"))
	return b.String()
}
