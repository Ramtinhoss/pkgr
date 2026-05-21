package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func addConfigCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{Use: "config", Short: "Inspect or edit config"}

	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print config path",
		RunE: func(c *cobra.Command, _ []string) error {
			base, _ := os.UserConfigDir()
			fmt.Fprintln(c.OutOrStdout(), filepath.Join(base, "pkgr", "config.toml"))
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Print effective config",
		RunE: func(c *cobra.Command, _ []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()
			fmt.Fprintf(c.OutOrStdout(), "%+v\n", app.Cfg)
			return nil
		},
	})
	root.AddCommand(cmd)
}
