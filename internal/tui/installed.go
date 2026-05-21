package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

// column indices for installed table (marker prepended)
const (
	iColMarker = 0
	iColName   = 1
	iColVer    = 2
	iColPM     = 3
)

type InstalledScreen struct {
	svc      *Services
	tbl      table.Model
	pmFilter string
	pkgs     []manager.Package // backing slice to map row idx → Package
	sel      Selection
}

func NewInstalledScreen(svc *Services) *InstalledScreen {
	cols := []table.Column{
		{Title: " ", Width: 2},
		{Title: "NAME", Width: 24},
		{Title: "VERSION", Width: 14},
		{Title: "PM", Width: 8},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &InstalledScreen{svc: svc, tbl: t, sel: NewSelection()}
}

func NewInstalledScreenFiltered(svc *Services, pmID string) *InstalledScreen {
	cols := []table.Column{
		{Title: " ", Width: 2},
		{Title: "NAME", Width: 24},
		{Title: "VERSION", Width: 14},
		{Title: "PM", Width: 8},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &InstalledScreen{svc: svc, tbl: t, pmFilter: pmID, sel: NewSelection()}
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
		s.pkgs = nil
		for _, p := range m.pkgs {
			if s.pmFilter != "" && p.Manager != s.pmFilter {
				continue
			}
			s.pkgs = append(s.pkgs, p)
		}
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

		case key.Matches(m, s.svc.Keys.Remove):
			if s.sel.Count() > 0 {
				return s, requestRemoveMany(s.svc, s.sel)
			}
			if pkg.Name != "" {
				return s, requestRemove(s.svc, pkg.Manager, pkg.Name)
			}

		case key.Matches(m, s.svc.Keys.Update):
			if s.sel.Count() > 0 {
				return s, requestUpdateMany(s.svc, s.sel)
			}
			if pkg.Name != "" {
				return s, requestUpdate(s.svc, pkg.Manager, pkg.Name)
			}
		}
	}

	var c tea.Cmd
	s.tbl, c = s.tbl.Update(msg)
	return s, c
}

func (s *InstalledScreen) pkgFromRow(row []string) manager.Package {
	if len(row) <= iColPM {
		return manager.Package{}
	}
	name := row[iColName]
	pm := row[iColPM]
	for _, p := range s.pkgs {
		if p.Name == name && p.Manager == pm {
			return p
		}
	}
	return manager.Package{Name: name, Manager: pm}
}

func (s *InstalledScreen) refreshTable() {
	rows := make([]table.Row, 0, len(s.pkgs))
	for _, p := range s.pkgs {
		marker := " "
		if s.sel.Has(p) {
			marker = "✓"
		}
		rows = append(rows, table.Row{marker, p.Name, p.Version, p.Manager})
	}
	s.tbl.SetRows(rows)
}

func (s *InstalledScreen) View() string {
	var b strings.Builder
	title := "Installed"
	if s.pmFilter != "" {
		title = "Installed (" + s.pmFilter + ")"
	}
	b.WriteString(s.svc.Theme.Title.Render(title))
	if s.sel.Count() > 0 {
		b.WriteString(fmt.Sprintf("  %s", s.svc.Theme.Warning.Render(fmt.Sprintf("%d selected", s.sel.Count()))))
	}
	b.WriteString("\n\n")
	b.WriteString(s.tbl.View())
	b.WriteString("\n")
	b.WriteString(s.svc.Theme.Subtle.Render("[r] remove  [u] update  [space] toggle  [a] sel-all  [A] clear  [esc] back"))
	return b.String()
}

// requestUpdate opens the confirm modal for a single package update.
func requestUpdate(svc *Services, pmID, name string) tea.Cmd {
	return func() tea.Msg {
		return PushScreenMsg{Screen: NewConfirmScreen(svc, ConfirmConfig{
			Title: "Update " + name,
			Body:  fmt.Sprintf("Run: %s update %s ?", pmID, name),
			OnYes: func() tea.Cmd {
				m, ok := svc.Reg.Get(pmID)
				if !ok {
					return nil
				}
				return func() tea.Msg {
					err := m.Update(svc.Ctx, name)
					return OpDoneMsg{Manager: pmID, Op: manager.OpUpdate, Err: err}
				}
			},
		})}
	}
}

// requestRemoveMany opens a confirm modal for bulk removal.
func requestRemoveMany(svc *Services, sel Selection) tea.Cmd {
	by := sel.GroupByPM()
	title := fmt.Sprintf("Remove %d packages", sel.Count())
	var lines []string
	for pm, names := range by {
		lines = append(lines, fmt.Sprintf("  %s uninstall %s", pm, strings.Join(names, " ")))
	}
	body := "Run:\n" + strings.Join(lines, "\n")
	return func() tea.Msg {
		return PushScreenMsg{Screen: NewConfirmScreen(svc, ConfirmConfig{
			Title: title,
			Body:  body,
			OnYes: func() tea.Cmd {
				return func() tea.Msg {
					var firstErr error
					for pm, names := range by {
						m, ok := svc.Reg.Get(pm)
						if !ok {
							continue
						}
						if err := m.Uninstall(svc.Ctx, names...); err != nil && firstErr == nil {
							firstErr = err
						}
					}
					return OpDoneMsg{Manager: "(multi)", Op: manager.OpUninstall, Err: firstErr}
				}
			},
		})}
	}
}

// requestUpdateMany opens a confirm modal for bulk update.
func requestUpdateMany(svc *Services, sel Selection) tea.Cmd {
	by := sel.GroupByPM()
	title := fmt.Sprintf("Update %d packages", sel.Count())
	var lines []string
	for pm, names := range by {
		lines = append(lines, fmt.Sprintf("  %s update %s", pm, strings.Join(names, " ")))
	}
	body := "Run:\n" + strings.Join(lines, "\n")
	return func() tea.Msg {
		return PushScreenMsg{Screen: NewConfirmScreen(svc, ConfirmConfig{
			Title: title,
			Body:  body,
			OnYes: func() tea.Cmd {
				return func() tea.Msg {
					var firstErr error
					for pm, names := range by {
						m, ok := svc.Reg.Get(pm)
						if !ok {
							continue
						}
						if err := m.Update(svc.Ctx, names...); err != nil && firstErr == nil {
							firstErr = err
						}
					}
					return OpDoneMsg{Manager: "(multi)", Op: manager.OpUpdate, Err: firstErr}
				}
			},
		})}
	}
}
