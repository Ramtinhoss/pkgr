package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/format"
	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/spec"
)

func addInstallCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "install <spec>...",
		Short: "Install one or more packages",
		Args:  cobra.MinimumNArgs(1),
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
				pm := parsed.PM
				if pm == "" {
					pm, err = resolvePM(cmd.Context(), app, parsed.Name, flags.Yes)
					if err != nil {
						return err
					}
				}
				byPM[pm] = append(byPM[pm], parsed.Name)
			}

			for pm, names := range byPM {
				m, ok := app.Reg.Get(pm)
				if !ok {
					return fmt.Errorf("unknown pm: %s", pm)
				}
				if err := m.Install(cmd.Context(), names...); err != nil {
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

// resolvePM picks the PM for a bare name. If multiple PMs return a hit,
// prompt the user unless yes=true, in which case pick by ranking.preferred.
func resolvePM(ctx context.Context, app *App, name string, yes bool) (string, error) {
	type cand struct{ m manager.Manager }
	var cands []cand
	for _, m := range app.Reg.Active() {
		if pkgs, err := m.Search(ctx, name); err == nil && len(pkgs) > 0 {
			cands = append(cands, cand{m: m})
		}
	}
	if len(cands) == 0 {
		return "", fmt.Errorf("no PM has %q", name)
	}
	if len(cands) == 1 || yes {
		// pick first by ranking.preferred order
		for _, p := range app.Cfg.Ranking.Preferred {
			for _, c := range cands {
				if c.m.ID() == p {
					return p, nil
				}
			}
		}
		return cands[0].m.ID(), nil
	}
	// interactive prompt
	fmt.Printf("Package %q exists in multiple PMs:\n", name)
	for i, c := range cands {
		fmt.Printf("  %d) %s\n", i+1, c.m.ID())
	}
	fmt.Print("Pick [1]: ")
	var pick int
	if _, err := fmt.Scanln(&pick); err != nil || pick < 1 || pick > len(cands) {
		pick = 1
	}
	return cands[pick-1].m.ID(), nil
}
