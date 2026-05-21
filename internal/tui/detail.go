package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type DetailScreen struct {
	svc *Services
	pkg manager.Package
}

func NewDetailScreen(svc *Services, p manager.Package) *DetailScreen {
	return &DetailScreen{svc: svc, pkg: p}
}

func (d *DetailScreen) Name() string  { return "detail" }
func (d *DetailScreen) Init() tea.Cmd { return nil }

func (d *DetailScreen) Update(msg tea.Msg) (Screen, tea.Cmd) {
	if m, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(m, d.svc.Keys.Install):
			return d, requestInstall(d.svc, d.pkg.Manager, d.pkg.Name)
		case key.Matches(m, d.svc.Keys.Remove):
			return d, requestRemove(d.svc, d.pkg.Manager, d.pkg.Name)
		case key.Matches(m, d.svc.Keys.ListInstalled):
			pmID := d.pkg.Manager
			return d, func() tea.Msg {
				return PushScreenMsg{Screen: NewInstalledScreenFiltered(d.svc, pmID)}
			}
		}
	}
	return d, nil
}

func (d *DetailScreen) View() string {
	var b strings.Builder
	b.WriteString(d.svc.Theme.Title.Render("Detail: " + d.pkg.Name))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Manager:     %s\n", d.pkg.Manager)
	fmt.Fprintf(&b, "Version:     %s\n", d.pkg.Version)
	if d.pkg.Latest != "" && d.pkg.Latest != d.pkg.Version {
		fmt.Fprintf(&b, "Latest:      %s\n", d.pkg.Latest)
	}
	fmt.Fprintf(&b, "Installed:   %v\n", d.pkg.Installed)
	if d.pkg.Homepage != "" {
		fmt.Fprintf(&b, "Homepage:    %s\n", d.pkg.Homepage)
	}
	if d.pkg.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", d.pkg.Description)
	}
	b.WriteString("\n")
	b.WriteString(d.svc.Theme.Subtle.Render("[i] install  [r] remove  [l] list installed from " + d.pkg.Manager + "  [esc] back"))
	return b.String()
}

func requestRemove(svc *Services, pmID, name string) tea.Cmd {
	return func() tea.Msg {
		return PushScreenMsg{Screen: NewConfirmScreen(svc, ConfirmConfig{
			Title: "Remove " + name,
			Body:  fmt.Sprintf("Run: %s remove %s ?", pmID, name),
			OnYes: func() tea.Cmd {
				m, ok := svc.Reg.Get(pmID)
				if !ok { return nil }
				return func() tea.Msg {
					err := m.Uninstall(svc.Ctx, name)
					return OpDoneMsg{Manager: pmID, Op: manager.OpUninstall, Err: err}
				}
			},
		})}
	}
}
