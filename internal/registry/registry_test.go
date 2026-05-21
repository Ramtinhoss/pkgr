package registry

import (
	"context"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

// Stub implements manager.Manager for testing
type Stub struct {
	id string
	d  bool
}

func (s *Stub) ID() string                                    { return s.id }
func (s *Stub) DisplayName() string                          { return s.id }
func (s *Stub) OSes() []manager.OS                           { return nil }
func (s *Stub) Detect() bool                                 { return s.d }
func (s *Stub) NeedsSudo(op manager.Op) bool                 { return false }
func (s *Stub) Scope() manager.Scope                         { return manager.ScopeSystem }
func (s *Stub) List(ctx context.Context) ([]manager.Package, error) {
	return nil, nil
}
func (s *Stub) Outdated(ctx context.Context) ([]manager.Package, error) {
	return nil, nil
}
func (s *Stub) Search(ctx context.Context, q string) ([]manager.Package, error) {
	return nil, nil
}
func (s *Stub) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{}, nil
}
func (s *Stub) Install(ctx context.Context, names ...string) error {
	return nil
}
func (s *Stub) Uninstall(ctx context.Context, names ...string) error {
	return nil
}
func (s *Stub) Update(ctx context.Context, names ...string) error {
	return nil
}

// TestActiveReturnsDetectedAndEnabled verifies Active() returns only managers
// where Detect()==true AND enabled doesn't override to false
func TestActiveReturnsDetectedAndEnabled(t *testing.T) {
	r := New()

	// Register 2 managers, both detected
	m1 := &Stub{id: "m1", d: true}
	m2 := &Stub{id: "m2", d: true}
	r.Register(m1)
	r.Register(m2)

	// SetEnabled disables m2 (false overrides detection)
	r.SetEnabled(map[string]bool{
		"m1": true,
		"m2": false,
	})

	// Active() should return only m1
	active := r.Active()
	if len(active) != 1 {
		t.Fatalf("Expected 1 active manager, got %d", len(active))
	}
	if active[0].ID() != "m1" {
		t.Fatalf("Expected m1, got %s", active[0].ID())
	}
}

// TestAllReturnsEverythingRegardlessOfDetection verifies All() returns
// all registered managers regardless of detection or enabled state
func TestAllReturnsEverythingRegardlessOfDetection(t *testing.T) {
	r := New()

	// Register 2 managers, only m1 detected
	m1 := &Stub{id: "m1", d: true}
	m2 := &Stub{id: "m2", d: false}
	r.Register(m1)
	r.Register(m2)

	// SetEnabled disables m1
	r.SetEnabled(map[string]bool{"m1": false})

	// All() should return both regardless
	all := r.All()
	if len(all) != 2 {
		t.Fatalf("Expected 2 managers, got %d", len(all))
	}
}
