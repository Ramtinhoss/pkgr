package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/ramtinhoss/pkgr/internal/update"
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
	err := root.Execute()
	maybeNagAboutUpdate(version)
	if err != nil {
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
	addSearchCmd(root, flags)
	addListCmd(root, flags)
	addOutdatedCmd(root, flags)
	// TODO: Tasks 10-12 add these subcommands
	addInfoCmd(root, flags)
	addInstallCmd(root, flags)
	addRemoveCmd(root, flags)
	addUpdateCmd(root, flags)
	// addOutdatedCmd(root, flags)
	addPMCmd(root, flags)
	addCacheCmd(root, flags)
	addDoctorCmd(root, flags)
	addConfigCmd(root, flags)
	addCompletionCmd(root)
	addTUICmd(root, flags)

	root.RunE = func(c *cobra.Command, args []string) error {
		c.SetArgs([]string{"tui"})
		return c.Execute()
	}
	return root
}

// maybeNagAboutUpdate checks for a newer release at most once per 24 hours
// and prints a one-line hint to stderr when a newer version is available.
// Opt-out: set update_check = false in config (checked via app.Cfg, but here
// we read the build-time version and rely on the caller to gate on cfg).
func maybeNagAboutUpdate(have string) {
	// Gate: stamp file persists the last check time.
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return
	}
	stampPath := filepath.Join(cacheDir, "pkgr", "last_update_check")
	if info, statErr := os.Stat(stampPath); statErr == nil && time.Since(info.ModTime()) < 24*time.Hour {
		return
	}
	_ = os.MkdirAll(filepath.Dir(stampPath), 0o755)
	_ = os.WriteFile(stampPath, []byte("ok"), 0o644)

	c := &update.Checker{}
	latest, err := c.Latest()
	if err != nil {
		return
	}
	if update.IsNewer(have, latest) {
		fmt.Fprintf(os.Stderr, "→ pkgr %s available (you have %s). https://github.com/ramtinhoss/pkgr/releases\n", latest, have)
	}
}
