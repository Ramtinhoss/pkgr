package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
	"github.com/ramtinhoss/pkgr/internal/spec"
)

func addUpdateCmd(root *cobra.Command, flags *rootFlags) {
	var all bool
	cmd := &cobra.Command{
		Use:   "update [spec]...",
		Short: "Update packages. No args + --all updates everything.",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil {
				return err
			}
			defer app.Closer()

			if len(args) == 0 {
				if !all {
					return fmt.Errorf("no specs given; use --all to update everything")
				}
				for _, m := range app.Reg.Active() {
					if err := m.Update(cmd.Context()); err != nil {
						fmt.Fprintln(cmd.ErrOrStderr(), err)
					}
					_ = app.Cache.Invalidate(m.ID() + "/installed")
					_ = app.Cache.Invalidate(m.ID() + "/outdated")
				}
				if flags.JSON {
					return format.JSONResult(cmd.OutOrStdout(), nil, nil)
				}
				return nil
			}

			byPM := make(map[string][]string)
			for _, s := range args {
				parsed, err := spec.Parse(s)
				if err != nil {
					return err
				}
				if parsed.PM == "" {
					return fmt.Errorf("update requires explicit @pm: %q", s)
				}
				byPM[parsed.PM] = append(byPM[parsed.PM], parsed.Name)
			}
			for pm, names := range byPM {
				m, ok := app.Reg.Get(pm)
				if !ok {
					return fmt.Errorf("unknown pm: %s", pm)
				}
				if err := m.Update(cmd.Context(), names...); err != nil {
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
	cmd.Flags().BoolVar(&all, "all", false, "update everything across all PMs")
	root.AddCommand(cmd)
}
