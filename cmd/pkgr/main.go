package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// build-time stamped values
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type buildInfo struct {
	Version, Commit, Date string
}

func main() {
	root := newRootCmd(buildInfo{Version: version, Commit: commit, Date: date})
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func newRootCmd(b buildInfo) *cobra.Command {
	root := &cobra.Command{
		Use:           "pkgr",
		Short:         "Cross-platform package manager TUI/CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd(b))
	return root
}
