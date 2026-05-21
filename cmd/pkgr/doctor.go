package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func addDoctorCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose adapter health",
		RunE: func(c *cobra.Command, _ []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()
			for _, m := range app.Reg.All() {
				status := "ok"
				if !m.Detect() { status = "binary not found" }
				fmt.Fprintf(c.OutOrStdout(), "%-10s %s\n", m.ID(), status)
			}
			return nil
		},
	}
	root.AddCommand(cmd)
}
