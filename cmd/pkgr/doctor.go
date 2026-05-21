package main

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

type doctorRow struct {
	Adapter  string `json:"adapter"`
	Detected bool   `json:"detected"`
	PingMS   int64  `json:"ping_ms"`
	Notes    string `json:"notes,omitempty"`
}

type doctorReport struct {
	Platform string      `json:"platform"`
	Theme    string      `json:"theme"`
	Adapters []doctorRow `json:"adapters"`
}

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

			rows := make([]doctorRow, 0, len(app.Reg.All()))
			for _, m := range app.Reg.All() {
				row := doctorRow{Adapter: m.ID(), Detected: m.Detect()}
				if m.Detect() {
					start := time.Now()
					_, listErr := m.List(ctx)
					row.PingMS = time.Since(start).Milliseconds()
					if listErr != nil {
						row.Notes = listErr.Error()
					}
				}
				rows = append(rows, row)
			}

			if flags.JSON {
				rep := doctorReport{
					Platform: runtime.GOOS + "/" + runtime.GOARCH,
					Theme:    app.Cfg.General.Theme,
					Adapters: rows,
				}
				enc := json.NewEncoder(c.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(rep)
			}

			tw := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintf(tw, "platform\t%s/%s\n", runtime.GOOS, runtime.GOARCH)
			fmt.Fprintf(tw, "config\t%s\n", app.Cfg.General.Theme)
			fmt.Fprintln(tw, "")
			fmt.Fprintln(tw, "ADAPTER\tDETECTED\tPING\tNOTES")
			for _, row := range rows {
				det := "no"
				if row.Detected {
					det = "yes"
				}
				ping := "-"
				if row.Detected {
					ping = fmt.Sprintf("%dms", row.PingMS)
				}
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", row.Adapter, det, ping, row.Notes)
			}
			return tw.Flush()
		},
	}
	root.AddCommand(cmd)
}
