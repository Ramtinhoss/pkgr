package config

import (
	"fmt"
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// Config represents the complete application configuration.
type Config struct {
	General  General            `toml:"general"`
	Cache    Cache              `toml:"cache"`
	Ranking  Ranking            `toml:"ranking"`
	Managers map[string]Manager `toml:"managers"`
}

// General contains general application settings.
type General struct {
	DefaultAssumeYes bool   `toml:"default_assume_yes"`
	Verbose         bool   `toml:"verbose"`
	Theme           string `toml:"theme"`
	JSONOutput      bool   `toml:"json_output"`
	UpdateCheck     bool   `toml:"update_check"`
}

// Cache contains cache configuration.
type Cache struct {
	Enabled      bool          `toml:"enabled"`
	InstalledTTL time.Duration `toml:"installed_ttl"`
	OutdatedTTL  time.Duration `toml:"outdated_ttl"`
	SearchTTL    time.Duration `toml:"search_ttl"`
	InfoTTL      time.Duration `toml:"info_ttl"`
	RegistryTTL  time.Duration `toml:"registry_ttl"`
}

// Ranking contains package manager ranking settings.
type Ranking struct {
	Preferred []string `toml:"preferred"`
}

// Manager contains per-manager configuration.
type Manager struct {
	Enabled   bool     `toml:"enabled"`
	ExtraArgs []string `toml:"extra_args"`
	Sudo      *bool    `toml:"sudo"`
}

// Load reads and parses a TOML configuration file, returning defaults if the file does not exist.
func Load(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// File missing; return defaults
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse TOML: %w", err)
	}

	// Re-apply defaults for zero-valued duration fields
	defaults := Defaults()
	if cfg.Cache.InstalledTTL == 0 {
		cfg.Cache.InstalledTTL = defaults.Cache.InstalledTTL
	}
	if cfg.Cache.OutdatedTTL == 0 {
		cfg.Cache.OutdatedTTL = defaults.Cache.OutdatedTTL
	}
	if cfg.Cache.SearchTTL == 0 {
		cfg.Cache.SearchTTL = defaults.Cache.SearchTTL
	}
	if cfg.Cache.InfoTTL == 0 {
		cfg.Cache.InfoTTL = defaults.Cache.InfoTTL
	}
	if cfg.Cache.RegistryTTL == 0 {
		cfg.Cache.RegistryTTL = defaults.Cache.RegistryTTL
	}

	// Re-apply Theme default if empty
	if cfg.General.Theme == "" {
		cfg.General.Theme = defaults.General.Theme
	}

	// Initialize Managers map if nil
	if cfg.Managers == nil {
		cfg.Managers = make(map[string]Manager)
	}

	return cfg, nil
}
