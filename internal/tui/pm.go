package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
)

type PMScreen struct {
	svc *Services
	tbl table.Model
}

func NewPMScreen(svc *Services) *PMScreen {
	cols := []table.Column{
		{Title: "ID", Width: 12},
		{Title: "DETECTED", Width: 10},
		{Title: "SCOPE", Width: 14},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	return &PMScreen{svc: svc, tbl: t}
}

func (p *PMScreen) Name() string { return "pm" }

func (p *PMScreen) Init() tea.Cmd {
	rows := []table.Row{}
	for _, m := range p.svc.Reg.All() {
		rows = append(rows, table.Row{m.ID(), fmt.Sprintf("%v", m.Detect()), string(m.Scope())})
	}
	p.tbl.SetRows(rows)
	return nil
}

func (p *PMScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	var c tea.Cmd
	p.tbl, c = p.tbl.Update(msg)
	return p, c
}

func (p *PMScreen) View() string {
	var b strings.Builder
	b.WriteString(p.svc.Theme.Title.Render("Package Managers"))
	b.WriteString("\n\n")
	b.WriteString(p.tbl.View())
	return b.String()
}
