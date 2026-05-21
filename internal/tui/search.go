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

type SearchScreen struct {
	svc      *Services
	input    textinput.Model
	spin     spinner.Model
	tbl      table.Model
	results  []orchestrator.Result
	pending  map[string]bool // active PMs
	query    string
	debounce time.Time
}

func NewSearchScreen(svc *Services) *SearchScreen {
	ti := textinput.New()
	ti.Placeholder = "search packages…"
	ti.Focus()
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	cols := []table.Column{
		{Title: "NAME", Width: 24},
		{Title: "VERSION", Width: 12},
		{Title: "PM", Width: 8},
		{Title: "INST", Width: 4},
		{Title: "SUMMARY", Width: 40},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &SearchScreen{svc: svc, input: ti, spin: sp, tbl: t, pending: map[string]bool{}}
}

func (s *SearchScreen) Name() string  { return "search" }
func (s *SearchScreen) Init() tea.Cmd { return tea.Batch(textinput.Blink, s.spin.Tick) }

func (s *SearchScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	var cmds []tea.Cmd

	switch m := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(m, s.svc.Keys.Install) && len(s.tbl.SelectedRow()) > 0 {
			name := s.tbl.SelectedRow()[0]
			pm := s.tbl.SelectedRow()[2]
			return s, requestInstall(s.svc, pm, name)
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
		inst := "no"
		if r.Pkg.Installed {
			inst = "yes"
		}
		rows = append(rows, table.Row{r.Pkg.Name, r.Pkg.Version, r.Pkg.Manager, inst, r.Pkg.Description})
	}
	s.tbl.SetRows(rows)
}

func (s *SearchScreen) View() string {
	var b strings.Builder
	b.WriteString(s.svc.Theme.Title.Render("Search"))
	b.WriteString("  ")
	b.WriteString(s.input.View())
	b.WriteString("\n\n")
	b.WriteString(s.tbl.View())
	if len(s.pending) > 0 {
		b.WriteString("\n")
		b.WriteString(s.spin.View())
		fmt.Fprintf(&b, " searching %d PMs…", len(s.pending))
	}
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

// TODO: replace with real impl in T7
type ConfirmConfig struct {
	Title string
	Body  string
	Sudo  bool
	OnYes func() tea.Cmd
}

func NewConfirmScreen(svc *Services, cfg ConfirmConfig) Screen {
	return &stubConfirmScreen{cfg: cfg}
}

type stubConfirmScreen struct{ cfg ConfirmConfig }

func (s *stubConfirmScreen) Name() string                         { return "confirm-stub" }
func (s *stubConfirmScreen) Init() tea.Cmd                        { return nil }
func (s *stubConfirmScreen) Update(msg tea.Msg) (Screen, tea.Cmd) { return s, nil }
func (s *stubConfirmScreen) View() string                         { return "confirm: " + s.cfg.Title }
