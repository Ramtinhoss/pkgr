package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
	"github.com/ramtinhoss/pkgr/internal/spec"
)

func addRemoveCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:     "remove <spec>...",
		Aliases: []string{"uninstall", "rm"},
		Short:   "Uninstall one or more packages",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil {
				return err
			}
			defer app.Closer()

			byPM := make(map[string][]string)
			for _, s := range args {
				parsed, err := spec.Parse(s)
				if err != nil {
					return err
				}
				if parsed.PM == "" {
					return fmt.Errorf("remove requires explicit @pm: %q", s)
				}
				byPM[parsed.PM] = append(byPM[parsed.PM], parsed.Name)
			}
			for pm, names := range byPM {
				m, ok := app.Reg.Get(pm)
				if !ok {
					return fmt.Errorf("unknown pm: %s", pm)
				}
				if err := m.Uninstall(cmd.Context(), names...); err != nil {
					return err
				}
				_ = app.Cache.Invalidate(pm + "/installed")
				_ = app.Cache.Invalidate(pm + "/outdated")
			}
			if flags.JSON {
				return format.JSONResult(cmd.OutOrStdout(), nil, nil)
			}
			return nil
		},
	}
	root.AddCommand(cmd)
}
