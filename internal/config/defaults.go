package config

import "time"

// Defaults returns a Config with sensible default values.
func Defaults() Config {
	return Config{
		General: General{
			DefaultAssumeYes: false,
			Verbose:         false,
			Theme:           "auto",
			JSONOutput:      false,
			UpdateCheck:     true,
		},
		Cache: Cache{
			Enabled:     true,
			InstalledTTL: 5 * time.Minute,
			OutdatedTTL:  30 * time.Minute,
			SearchTTL:    1 * time.Hour,
			InfoTTL:      24 * time.Hour,
			RegistryTTL:   1 * time.Hour,
		},
		Ranking: Ranking{
			Preferred: []string{
				"brew", "apt", "dnf", "pacman", "winget", "scoop",
				"uv", "pipx", "pip", "cargo", "npm", "pnpm", "bun",
			},
		},
		Managers: make(map[string]Manager),
	}
}
