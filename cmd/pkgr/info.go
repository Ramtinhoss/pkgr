package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/spec"
)

func addInfoCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "info <spec>",
		Short: "Show details for a package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil {
				return err
			}
			defer app.Closer()

			s, err := spec.Parse(args[0])
			if err != nil {
				return err
			}

			mgrs := app.Reg.Active()
			if s.PM != "" {
				m, ok := app.Reg.Get(s.PM)
				if !ok {
					return fmt.Errorf("unknown pm: %s", s.PM)
				}
				mgrs = []manager.Manager{m}
			} else if len(flags.PMs) > 0 {
				mgrs = filterPMs(mgrs, flags.PMs)
			}

			for _, m := range mgrs {
				p, err := m.Info(cmd.Context(), s.Name)
				if err == nil {
					if flags.JSON {
						return format.JSONResult(cmd.OutOrStdout(), []manager.Package{p}, nil)
					}
					return format.HumanInfo(cmd.OutOrStdout(), p)
				}
			}
			return fmt.Errorf("not found: %s", s.Name)
		},
	}
	root.AddCommand(cmd)
}
