package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

type Keys struct {
	Quit          key.Binding
	Help          key.Binding
	Back          key.Binding
	Search        key.Binding
	Install       key.Binding
	Remove        key.Binding
	Update        key.Binding
	Outdated      key.Binding
	Palette       key.Binding
	Log           key.Binding
	Cancel        key.Binding
	ListInstalled key.Binding
}

func DefaultKeys() Keys {
	return Keys{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Install: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "install"),
		),
		Remove: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "remove"),
		),
		Update: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "update"),
		),
		Outdated: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "outdated"),
		),
		Palette: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "palette"),
		),
		Log: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "log"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "cancel"),
		),
		ListInstalled: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "list installed from PM"),
		),
	}
}
