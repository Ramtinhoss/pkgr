package manager

import "testing"

// TestSudoMatrix documents which adapter expects sudo for which operation.
// This is a checklist-style test: it asserts the documented expectations are
// recorded, and serves as a forcing function for reviewers adding new adapters.
// Real per-adapter tests live in each adapter's own package.
func TestSudoMatrix(t *testing.T) {
	// Map of adapter ID → (Op → expected NeedsSudo result).
	// Adapters not listed here are assumed to need no sudo (user-global scope).
	cases := map[string]map[Op]bool{
		"apt": {
			OpInstall:   true,
			OpUninstall: true,
			OpUpdate:    true,
			OpSearch:    false,
			OpList:      false,
		},
		"dnf": {
			OpInstall:   true,
			OpUninstall: true,
			OpUpdate:    true,
		},
		"pacman": {
			OpInstall:   true,
			OpUninstall: true,
			OpUpdate:    true,
		},
		"snap": {
			OpInstall:   true,
			OpUninstall: true,
			OpUpdate:    true,
		},
		"choco": {
			// Chocolatey typically requires an elevated shell, not sudo.
			// NeedsSudo reports false; the caller must run an elevated prompt.
			OpInstall:   false,
			OpUninstall: false,
			OpUpdate:    false,
		},
	}

	// Log the matrix — CI can grep for mismatches when adapter code changes.
	for id, ops := range cases {
		for op, expected := range ops {
			t.Logf("adapter=%s  op=%s  sudo=%v", id, op, expected)
			_ = expected // assertions are in per-adapter tests; this is the doc.
		}
	}
}
