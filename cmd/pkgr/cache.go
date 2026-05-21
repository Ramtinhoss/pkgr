package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func addCacheCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{Use: "cache", Short: "Manage local cache"}
	cmd.AddCommand(&cobra.Command{
		Use:   "clear [pm]",
		Short: "Clear cache (all PMs or one)",
		RunE: func(c *cobra.Command, args []string) error {
			app, err := newApp(*flags)
			if err != nil { return err }
			defer app.Closer()
			target := app.Cache.Root
			if len(args) == 1 {
				target = filepath.Join(app.Cache.Root, args[0])
			}
			fmt.Fprintf(c.OutOrStdout(), "removing %s\n", target)
			return os.RemoveAll(target)
		},
	})
	root.AddCommand(cmd)
}
