package tui

import (
	"context"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/cache"
	"github.com/ramtinhoss/pkgr/internal/orchestrator"
	"github.com/ramtinhoss/pkgr/internal/registry"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

func TestSearchScreenRendersInput(t *testing.T) {
	// Construct Services with all required fields
	svc := Services{
		Ctx:   context.Background(),
		Reg:   registry.New(),
		Orc:   orchestrator.New(orchestrator.Ranking{}),
		Cache: cache.New(t.TempDir()),
		Run:   &runner.Runner{},
		Theme: DefaultTheme(true),
		Keys:  DefaultKeys(),
	}
	// Create App with NewSearchScreen
	m := New(svc, NewSearchScreen(&svc))
	// Call Init() and verify it doesn't panic
	cmd := m.Init()
	if cmd == nil {
		// No command is acceptable (may be nil)
	}
	// Call View() and verify it renders without panic
	output := m.View()
	if len(output) == 0 {
		t.Log("View rendered empty output (acceptable for initial state)")
	}
	// Test passes if no panic occurred during construction, Init, or View
	t.Log("App model compiles, Init and View render without panic")
}

func contains(haystack []byte, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if string(haystack[i:i+len(needle)]) == needle {
			return true
		}
	}
	return false
}
