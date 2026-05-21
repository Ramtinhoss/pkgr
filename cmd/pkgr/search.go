package main

import (
	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
	"github.com/ramtinhoss/pkgr/internal/manager"
)

func addSearchCmd(root *cobra.Command, flags *rootFlags) {
	var limit int
	var installedOnly bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search packages across all detected PMs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil {
				return err
			}
			defer app.Closer()

			mgrs := app.Reg.Active()
			if len(flags.PMs) > 0 {
				mgrs = filterPMs(mgrs, flags.PMs)
			}
			results, errs := app.Orc.Search(cmd.Context(), mgrs, args[0])

			pkgs := make([]manager.Package, 0, len(results))
			for _, r := range results {
				if installedOnly && !r.Pkg.Installed {
					continue
				}
				pkgs = append(pkgs, r.Pkg)
				if limit > 0 && len(pkgs) >= limit {
					break
				}
			}

			if flags.JSON {
				return format.JSONResult(cmd.OutOrStdout(), pkgs, errs)
			}
			return format.HumanSearch(cmd.OutOrStdout(), pkgs)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "max results")
	cmd.Flags().BoolVar(&installedOnly, "installed-only", false, "only show installed pkgs")
	root.AddCommand(cmd)
}

func filterPMs(all []manager.Manager, ids []string) []manager.Manager {
	allow := make(map[string]bool, len(ids))
	for _, id := range ids {
		allow[id] = true
	}
	out := make([]manager.Manager, 0, len(all))
	for _, m := range all {
		if allow[m.ID()] {
			out = append(out, m)
		}
	}
	return out
}
