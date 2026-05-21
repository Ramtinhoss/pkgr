package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type ConfirmConfig struct {
	Title string
	Body  string
	Sudo  bool
	OnYes func() tea.Cmd
}

type ConfirmScreen struct {
	svc *Services
	cfg ConfirmConfig
}

func NewConfirmScreen(svc *Services, cfg ConfirmConfig) *ConfirmScreen {
	return &ConfirmScreen{svc: svc, cfg: cfg}
}

func (c *ConfirmScreen) Name() string  { return "confirm" }
func (c *ConfirmScreen) Init() tea.Cmd { return nil }

func (c *ConfirmScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if m, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(m, key.NewBinding(key.WithKeys("y", "Y"))):
			cmd := c.cfg.OnYes
			return c, tea.Sequence(
				func() tea.Msg { return PopScreenMsg{} },
				func() tea.Msg { return StatusMsg{Text: "running…"} },
				cmd(),
			)
		case key.Matches(m, key.NewBinding(key.WithKeys("n", "N", "esc"))):
			return c, func() tea.Msg { return PopScreenMsg{} }
		}
	}
	return c, nil
}

func (c *ConfirmScreen) View() string {
	var b strings.Builder
	hdr := c.svc.Theme.Title.Render(c.cfg.Title)
	if c.cfg.Sudo {
		hdr = c.svc.Theme.Warning.Render("REQUIRES SUDO — ") + hdr
	}
	b.WriteString(hdr)
	b.WriteString("\n\n")
	b.WriteString(c.cfg.Body)
	b.WriteString("\n\n[y] yes   [n/esc] no")
	if c.svc.Run != nil && c.svc.Run.DryRun {
		b.WriteString("\n\n")
		b.WriteString(c.svc.Theme.Subtle.Render("DRY-RUN: no command will execute"))
	}
	return b.String()
}
