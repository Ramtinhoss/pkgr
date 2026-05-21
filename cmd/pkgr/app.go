package main

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ramtinhoss/pkgr/internal/cache"
	"github.com/ramtinhoss/pkgr/internal/config"
	pkgrlog "github.com/ramtinhoss/pkgr/internal/log"
	"github.com/ramtinhoss/pkgr/internal/orchestrator"
	"github.com/ramtinhoss/pkgr/internal/registry"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type App struct {
	Cfg    config.Config
	Reg    *registry.Registry
	Orc    *orchestrator.Orchestrator
	Cache  *cache.Cache
	Run    *runner.Runner
	Log    *slog.Logger
	Closer func() error
}

func newApp(flags rootFlags) (*App, error) {
	cfgPath := flags.ConfigPath
	if cfgPath == "" {
		base, _ := os.UserConfigDir()
		cfgPath = filepath.Join(base, "pkgr", "config.toml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	stateDir, _ := os.UserHomeDir()
	logPath := filepath.Join(stateDir, ".local", "state", "pkgr", "pkgr.log")
	l, closer, err := pkgrlog.Setup(pkgrlog.Options{Path: logPath, Verbose: flags.Verbose || cfg.General.Verbose})
	if err != nil {
		return nil, err
	}

	cacheDir, _ := os.UserCacheDir()
	c := cache.New(filepath.Join(cacheDir, "pkgr"))

	r := &runner.Runner{DryRun: flags.DryRun, Out: os.Stdout}
	reg := registry.New()
	registerAdapters(reg, r)
	enabled := make(map[string]bool)
	for id, m := range cfg.Managers {
		enabled[id] = m.Enabled
	}
	reg.SetEnabled(enabled)

	orc := orchestrator.New(orchestrator.Ranking{Preferred: cfg.Ranking.Preferred})

	return &App{Cfg: cfg, Reg: reg, Orc: orc, Cache: c, Run: r, Log: l, Closer: closer}, nil
}
