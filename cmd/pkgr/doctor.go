package main

import (
	"context"
	"fmt"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

func addDoctorCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Deep adapter health checks",
		RunE: func(c *cobra.Command, _ []string) error {
			app, err := newApp(*flags)
			if err != nil {
				return err
			}
			defer app.Closer()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			tw := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "platform\t%s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintf(tw, "config\t%s\n", app.Cfg.General.Theme)
			fmt.Fprintln(tw, "")
			fmt.Fprintln(tw, "ADAPTER\tDETECTED\tPING\tNOTES")

			for _, m := range app.Reg.All() {
				det := "no"
				if m.Detect() {
					det = "yes"
				}
				ping := "-"
				notes := ""
				if m.Detect() {
					start := time.Now()
					_, err := m.List(ctx)
					dur := time.Since(start)
					if err != nil {
						notes = err.Error()
					}
					ping = fmt.Sprintf("%dms", dur.Milliseconds())
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", m.ID(), det, ping, notes)
			}
			return tw.Flush()
		},
	}
	root.AddCommand(cmd)
}
