package main

import "github.com/spf13/cobra"

type rootFlags struct {
	PMs        []string
	JSON       bool
	NoColor    bool
	Yes        bool
	DryRun     bool
	NoCache    bool
	Verbose    bool
	ConfigPath string
}

func bindRootFlags(cmd *cobra.Command, f *rootFlags) {
	cmd.PersistentFlags().StringSliceVar(&f.PMs, "pm", nil, "restrict to one or more PMs")
	cmd.PersistentFlags().BoolVar(&f.JSON, "json", false, "machine output")
	cmd.PersistentFlags().BoolVar(&f.NoColor, "no-color", false, "disable ANSI colors")
	cmd.PersistentFlags().BoolVarP(&f.Yes, "yes", "y", false, "auto-confirm prompts")
	cmd.PersistentFlags().BoolVar(&f.DryRun, "dry-run", false, "print commands without executing")
	cmd.PersistentFlags().BoolVar(&f.NoCache, "no-cache", false, "bypass cache")
	cmd.PersistentFlags().BoolVarP(&f.Verbose, "verbose", "v", false, "verbose logging to stderr")
	cmd.PersistentFlags().StringVar(&f.ConfigPath, "config", "", "alternative config file path")
}
