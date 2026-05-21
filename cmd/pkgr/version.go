package main

import (
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd(b buildInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build info",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Printf("pkgr %s\ncommit:  %s\ndate:    %s\ngo:      %s\nplatform: %s/%s\n",
				b.Version, b.Commit, b.Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		},
	}
}
