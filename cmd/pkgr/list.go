package main

import (
	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
)

func addListCmd(root *cobra.Command, flags *rootFlags) {
	var outdated bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed packages across PMs",
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()

			mgrs := app.Reg.Active()
			if len(flags.PMs) > 0 { mgrs = filterPMs(mgrs, flags.PMs) }

			if outdated {
				p, errs := app.Orc.Outdated(cmd.Context(), mgrs)
				if flags.JSON { return format.JSONResult(cmd.OutOrStdout(), p, errs) }
				return format.HumanList(cmd.OutOrStdout(), p)
			}
			p, errs := app.Orc.List(cmd.Context(), mgrs)
			if flags.JSON { return format.JSONResult(cmd.OutOrStdout(), p, errs) }
			return format.HumanList(cmd.OutOrStdout(), p)
		},
	}
	cmd.Flags().BoolVar(&outdated, "outdated", false, "show only outdated packages")
	root.AddCommand(cmd)
}
