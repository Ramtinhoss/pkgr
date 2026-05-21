package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/orchestrator"
)

// column indices for search table (marker prepended)
const (
	sColMarker = 0
	sColName   = 1
	sColVer    = 2
	sColPM     = 3
	sColInst   = 4
	// sColSummary = 5  (unused for lookups)
)

type SearchScreen struct {
	svc      *Services
	input    textinput.Model
	spin     spinner.Model
	tbl      table.Model
	results  []orchestrator.Result
	pending  map[string]bool // active PMs
	query    string
	debounce time.Time
	sel      Selection
}

func NewSearchScreen(svc *Services) *SearchScreen {
	ti := textinput.New()
	ti.Placeholder = "search packages…"
	ti.Focus()
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	cols := []table.Column{
		{Title: " ", Width: 2},
		{Title: "NAME", Width: 24},
		{Title: "VERSION", Width: 12},
		{Title: "PM", Width: 8},
		{Title: "INST", Width: 4},
		{Title: "SUMMARY", Width: 40},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &SearchScreen{svc: svc, input: ti, spin: sp, tbl: t, pending: map[string]bool{}, sel: NewSelection()}
}

func (s *SearchScreen) Name() string  { return "search" }
func (s *SearchScreen) Init() tea.Cmd { return tea.Batch(textinput.Blink, s.spin.Tick) }

func (s *SearchScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.KeyMsg:
		row := s.tbl.SelectedRow()

		switch {
		case key.Matches(m, s.svc.Keys.ToggleSelect) && len(row) > sColPM:
			pkg := s.pkgFromRow(row)
			s.sel.Toggle(pkg)
			s.refreshTable()
			return s, tea.Batch(cmds...)

		case key.Matches(m, s.svc.Keys.SelectAll):
			for _, r := range s.results {
				s.sel[KeyFor(r.Pkg)] = r.Pkg
			}
			s.refreshTable()
			return s, tea.Batch(cmds...)

		case key.Matches(m, s.svc.Keys.ClearSelect):
			s.sel.Clear()
			s.refreshTable()
			return s, tea.Batch(cmds...)

		case key.Matches(m, s.svc.Keys.Install):
			if s.sel.Count() > 0 {
				return s, requestInstallMany(s.svc, s.sel)
			}
			if len(row) > sColPM {
				name := row[sColName]
				pm := row[sColPM]
				return s, requestInstall(s.svc, pm, name)
			}

		case key.Matches(m, s.svc.Keys.ListInstalled) && len(row) > sColPM:
			pmID := row[sColPM]
			return s, func() tea.Msg {
				return PushScreenMsg{Screen: NewInstalledScreenFiltered(s.svc, pmID)}
			}
		}

	case spinner.TickMsg:
		var c tea.Cmd
		s.spin, c = s.spin.Update(m)
		cmds = append(cmds, c)

	case SearchPartialMsg:
		for _, p := range m.Results {
			s.results = append(s.results, orchestrator.Result{Pkg: p})
		}
		delete(s.pending, m.Manager)
		s.refreshTable()

	case SearchDoneMsg:
		s.results = m.All
		s.pending = nil
		s.refreshTable()
	}

	var c tea.Cmd
	s.input, c = s.input.Update(msg)
	cmds = append(cmds, c)
	s.tbl, c = s.tbl.Update(msg)
	cmds = append(cmds, c)

	// debounce-trigger
	if v := s.input.Value(); v != s.query && time.Since(s.debounce) > 250*time.Millisecond {
		s.query = v
		s.debounce = time.Now()
		cmds = append(cmds, s.startSearch(v))
	}
	return s, tea.Batch(cmds...)
}

func (s *SearchScreen) pkgFromRow(row []string) manager.Package {
	if len(row) <= sColPM {
		return manager.Package{}
	}
	// Try to find the full Package in results by name+PM
	name := row[sColName]
	pm := row[sColPM]
	for _, r := range s.results {
		if r.Pkg.Name == name && r.Pkg.Manager == pm {
			return r.Pkg
		}
	}
	return manager.Package{Name: name, Manager: pm}
}

func (s *SearchScreen) startSearch(q string) tea.Cmd {
	if q == "" {
		return nil
	}
	mgrs := s.svc.Reg.Active()
	s.pending = make(map[string]bool, len(mgrs))
	for _, m := range mgrs {
		s.pending[m.ID()] = true
	}
	s.results = nil
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(s.svc.Ctx, 15*time.Second)
		defer cancel()
		res, errs := s.svc.Orc.Search(ctx, mgrs, q)
		return SearchDoneMsg{All: res, Errs: errs}
	}
}

func (s *SearchScreen) refreshTable() {
	rows := make([]table.Row, 0, len(s.results))
	for _, r := range s.results {
		marker := " "
		if s.sel.Has(r.Pkg) {
			marker = "✓"
		}
		inst := "no"
		if r.Pkg.Installed {
			inst = "yes"
		}
		rows = append(rows, table.Row{marker, r.Pkg.Name, r.Pkg.Version, r.Pkg.Manager, inst, r.Pkg.Description})
	}
	s.tbl.SetRows(rows)
}

func (s *SearchScreen) View() string {
	var b strings.Builder
	b.WriteString(s.svc.Theme.Title.Render("Search"))
	b.WriteString("  ")
	b.WriteString(s.input.View())
	if s.sel.Count() > 0 {
		b.WriteString(fmt.Sprintf("  %s", s.svc.Theme.Warning.Render(fmt.Sprintf("%d selected", s.sel.Count()))))
	}
	b.WriteString("\n\n")
	b.WriteString(s.tbl.View())
	if len(s.pending) > 0 {
		b.WriteString("\n")
		b.WriteString(s.spin.View())
		fmt.Fprintf(&b, " searching %d PMs…", len(s.pending))
	}
	b.WriteString("\n")
	b.WriteString(s.svc.Theme.Subtle.Render("[i] install  [l] installed from PM  [space] toggle  [a] sel-all  [A] clear  [esc] back"))
	return b.String()
}

// requestInstall opens the confirm modal for the chosen pkg.
func requestInstall(svc *Services, pmID, name string) tea.Cmd {
	return func() tea.Msg {
		return PushScreenMsg{Screen: NewConfirmScreen(svc, ConfirmConfig{
			Title: "Install " + name,
			Body:  fmt.Sprintf("Run: %s install %s ?", pmID, name),
			OnYes: func() tea.Cmd {
				m, ok := svc.Reg.Get(pmID)
				if !ok {
					return nil
				}
				return func() tea.Msg {
					err := m.Install(svc.Ctx, name)
					return OpDoneMsg{Manager: pmID, Op: manager.OpInstall, Err: err}
				}
			},
		})}
	}
}

// requestInstallMany opens a confirm modal for bulk install.
func requestInstallMany(svc *Services, sel Selection) tea.Cmd {
	by := sel.GroupByPM()
	title := fmt.Sprintf("Install %d packages", sel.Count())
	var lines []string
	for pm, names := range by {
		lines = append(lines, fmt.Sprintf("  %s install %s", pm, strings.Join(names, " ")))
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
						if err := m.Install(svc.Ctx, names...); err != nil && firstErr == nil {
							firstErr = err
						}
					}
					return OpDoneMsg{Manager: "(multi)", Op: manager.OpInstall, Err: firstErr}
				}
			},
		})}
	}
}
