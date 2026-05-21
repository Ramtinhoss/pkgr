package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaultsWhenFileMissing(t *testing.T) {
	// Test that Load() returns defaults when config file does not exist
	cfg, err := Load("/nonexistent/path/config.toml")
	if err != nil {
		t.Fatalf("Load() should not error when file missing: %v", err)
	}

	// Verify defaults are applied
	if cfg.General.Theme != "auto" {
		t.Errorf("expected Theme='auto', got %q", cfg.General.Theme)
	}
	if !cfg.General.UpdateCheck {
		t.Errorf("expected UpdateCheck=true, got false")
	}
	if cfg.Cache.InstalledTTL != 5*time.Minute {
		t.Errorf("expected InstalledTTL=5m, got %v", cfg.Cache.InstalledTTL)
	}
	if cfg.Cache.OutdatedTTL != 30*time.Minute {
		t.Errorf("expected OutdatedTTL=30m, got %v", cfg.Cache.OutdatedTTL)
	}
	if len(cfg.Ranking.Preferred) == 0 {
		t.Errorf("expected Ranking.Preferred to be populated, got empty")
	}
	if cfg.Managers == nil {
		t.Errorf("expected Managers to be initialized, got nil")
	}
}

func TestLoadParsesTOML(t *testing.T) {
	// Create temporary TOML file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	tomlContent := `[general]
default_assume_yes = true
verbose = true
theme = "dark"
json_output = true
update_check = false

[cache]
enabled = true
installed_ttl = "10m"
outdated_ttl = "45m"
search_ttl = "2h"
info_ttl = "48h"
registry_ttl = "2h"

[ranking]
preferred = ["brew", "apt", "custom"]

[managers.brew]
enabled = true
extra_args = ["--no-analytics"]
sudo = false

[managers.apt]
enabled = false
`

	if err := os.WriteFile(configPath, []byte(tomlContent), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify general settings override defaults
	if !cfg.General.DefaultAssumeYes {
		t.Errorf("expected DefaultAssumeYes=true, got false")
	}
	if !cfg.General.Verbose {
		t.Errorf("expected Verbose=true, got false")
	}
	if cfg.General.Theme != "dark" {
		t.Errorf("expected Theme='dark', got %q", cfg.General.Theme)
	}
	if !cfg.General.JSONOutput {
		t.Errorf("expected JSONOutput=true, got false")
	}
	if cfg.General.UpdateCheck {
		t.Errorf("expected UpdateCheck=false, got true")
	}

	// Verify cache settings override defaults
	if cfg.Cache.InstalledTTL != 10*time.Minute {
		t.Errorf("expected InstalledTTL=10m, got %v", cfg.Cache.InstalledTTL)
	}
	if cfg.Cache.OutdatedTTL != 45*time.Minute {
		t.Errorf("expected OutdatedTTL=45m, got %v", cfg.Cache.OutdatedTTL)
	}
	if cfg.Cache.SearchTTL != 2*time.Hour {
		t.Errorf("expected SearchTTL=2h, got %v", cfg.Cache.SearchTTL)
	}

	// Verify ranking settings override defaults
	if len(cfg.Ranking.Preferred) != 3 {
		t.Errorf("expected 3 preferred managers, got %d", len(cfg.Ranking.Preferred))
	}
	if cfg.Ranking.Preferred[0] != "brew" || cfg.Ranking.Preferred[2] != "custom" {
		t.Errorf("expected Ranking.Preferred to match TOML, got %v", cfg.Ranking.Preferred)
	}

	// Verify manager settings
	brewMgr, ok := cfg.Managers["brew"]
	if !ok {
		t.Fatalf("expected 'brew' manager in Managers map")
	}
	if !brewMgr.Enabled {
		t.Errorf("expected brew.Enabled=true, got false")
	}
	if len(brewMgr.ExtraArgs) != 1 || brewMgr.ExtraArgs[0] != "--no-analytics" {
		t.Errorf("expected brew.ExtraArgs=['--no-analytics'], got %v", brewMgr.ExtraArgs)
	}
	if brewMgr.Sudo == nil || *brewMgr.Sudo {
		t.Errorf("expected brew.Sudo=false, got %v", brewMgr.Sudo)
	}

	aptMgr, ok := cfg.Managers["apt"]
	if !ok {
		t.Fatalf("expected 'apt' manager in Managers map")
	}
	if aptMgr.Enabled {
		t.Errorf("expected apt.Enabled=false, got true")
	}
}
