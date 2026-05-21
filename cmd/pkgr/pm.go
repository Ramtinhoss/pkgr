package main

import (
	"encoding/json"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

type pmRow struct {
	ID       string `json:"id"`
	Scope    string `json:"scope"`
	Detected bool   `json:"detected"`
	Enabled  bool   `json:"enabled"`
}

func addPMCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "pm",
		Short: "Manage package-manager adapters",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List adapters",
		RunE: func(c *cobra.Command, _ []string) error {
			app, err := newApp(*flags)
			if err != nil {
				return err
			}
			defer app.Closer()

			if flags.JSON {
				rows := make([]pmRow, 0, len(app.Reg.All()))
				for _, m := range app.Reg.All() {
					enabled := true
					if v, ok := app.Cfg.Managers[m.ID()]; ok && !v.Enabled {
						enabled = false
					}
					rows = append(rows, pmRow{
						ID:       m.ID(),
						Scope:    string(m.Scope()),
						Detected: m.Detect(),
						Enabled:  enabled,
					})
				}
				enc := json.NewEncoder(c.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rows)
			}

			tw := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tDETECTED\tENABLED\tSCOPE")
			for _, m := range app.Reg.All() {
				en := "yes"
				if v, ok := app.Cfg.Managers[m.ID()]; ok && !v.Enabled {
					en = "no"
				}
				fmt.Fprintf(tw, "%s\t%v\t%s\t%s\n", m.ID(), m.Detect(), en, m.Scope())
			}
			return tw.Flush()
		},
	})
	root.AddCommand(cmd)
}
