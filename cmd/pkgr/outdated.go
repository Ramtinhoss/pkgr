package main

import "github.com/spf13/cobra"

func addOutdatedCmd(root *cobra.Command, flags *rootFlags) {
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Show outdated packages across PMs (alias for 'list --outdated')",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Re-dispatch through list cmd with the flag set.
			root.SetArgs(append([]string{"list", "--outdated"}, args...))
			return root.Execute()
		},
	}
	root.AddCommand(cmd)
}
