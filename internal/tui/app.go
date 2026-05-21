package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ramtinhoss/pkgr/internal/cache"
	"github.com/ramtinhoss/pkgr/internal/config"
	"github.com/ramtinhoss/pkgr/internal/orchestrator"
	"github.com/ramtinhoss/pkgr/internal/registry"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

// Services bundles all dependencies a screen may need.
type Services struct {
	Ctx   context.Context
	Cfg   config.Config
	Reg   *registry.Registry
	Orc   *orchestrator.Orchestrator
	Cache *cache.Cache
	Run   *runner.Runner
	Theme Theme
	Keys  Keys
}

type App struct {
	Svc    Services
	stack  []Screen
	status string
	width  int
	height int
}

func New(svc Services, initial Screen) *App {
	return &App{Svc: svc, stack: []Screen{initial}}
}

func (a *App) Init() tea.Cmd {
	return a.top().Init()
}

func (a *App) top() Screen { return a.stack[len(a.stack)-1] }

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = m.Width, m.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(m, a.Svc.Keys.Quit):
			return a, tea.Quit
		case key.Matches(m, a.Svc.Keys.Back):
			if len(a.stack) > 1 {
				a.stack = a.stack[:len(a.stack)-1]
				return a, a.top().Init()
			}
		}
	case PushScreenMsg:
		a.stack = append(a.stack, m.Screen)
		return a, m.Screen.Init()
	case PopScreenMsg:
		if len(a.stack) > 1 {
			a.stack = a.stack[:len(a.stack)-1]
		}
		return a, a.top().Init()
	case StatusMsg:
		a.status = m.Text
	}
	s, cmd := a.top().Update(msg)
	a.stack[len(a.stack)-1] = s
	return a, cmd
}

func (a *App) View() string {
	top := a.Svc.Theme.Title.Render("pkgr") + "  " + a.Svc.Theme.Subtle.Render(strings.Repeat("─", max(0, a.width-7)))
	status := a.Svc.Theme.StatusBar.Width(a.width).Render("status: " + a.status)
	help := a.Svc.Theme.HelpBar.Render("[/] search  [i] install  [u] update  [d] remove  [?] help  [q] quit")
	return top + "\n" + a.top().View() + "\n" + status + "\n" + help
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
