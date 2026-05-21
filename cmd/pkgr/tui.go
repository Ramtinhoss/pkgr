package main

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/tui"
)

func addTUICmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive TUI",
		RunE: func(c *cobra.Command, _ []string) error {
			app, err := newApp(*flags)
			if err != nil {
				return err
			}
			defer app.Closer()

			svc := tui.Services{
				Ctx:   context.Background(),
				Cfg:   app.Cfg,
				Reg:   app.Reg,
				Orc:   app.Orc,
				Cache: app.Cache,
				Run:   app.Run,
				Theme: tui.ResolveTheme(app.Cfg.General.Theme),
				Keys:  tui.DefaultKeys(),
			}
			home := tui.NewHomeScreen(&svc)
			m := tui.New(svc, home)
			p := tea.NewProgram(m, tea.WithAltScreen())
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("tui: %w", err)
			}
			return nil
		},
	}
	root.AddCommand(cmd)
}
