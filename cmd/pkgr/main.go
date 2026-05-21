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
	flags := &rootFlags{}
	root := &cobra.Command{
		Use:           "pkgr",
		Short:         "Cross-platform package manager TUI/CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	bindRootFlags(root, flags)
	root.AddCommand(newVersionCmd(b))
	// TODO: Tasks 8-12 add these subcommands
	// addSearchCmd(root, flags)
	// addListCmd(root, flags)
	// addInfoCmd(root, flags)
	// addInstallCmd(root, flags)
	// addRemoveCmd(root, flags)
	// addUpdateCmd(root, flags)
	// addOutdatedCmd(root, flags)
	// addPMCmd(root, flags)
	// addCacheCmd(root, flags)
	// addDoctorCmd(root, flags)
	// addConfigCmd(root, flags)
	// addCompletionCmd(root)
	return root
}
