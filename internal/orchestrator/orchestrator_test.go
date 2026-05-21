package orchestrator

import (
	"context"
	"errors"
	"sort"
	"testing"

	"github.com/ramtinhoss/pkgr/internal/manager"
)

type fakeMgr struct {
	id   string
	pkgs []manager.Package
	err  error
}

func (f *fakeMgr) ID() string                                                  { return f.id }
func (f *fakeMgr) DisplayName() string                                         { return f.id }
func (f *fakeMgr) OSes() []manager.OS                                          { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (f *fakeMgr) Detect() bool                                                { return true }
func (f *fakeMgr) NeedsSudo(manager.Op) bool                                   { return false }
func (f *fakeMgr) Scope() manager.Scope                                        { return manager.ScopeUserGlobal }
func (f *fakeMgr) List(context.Context) ([]manager.Package, error)             { return f.pkgs, f.err }
func (f *fakeMgr) Outdated(context.Context) ([]manager.Package, error)         { return f.pkgs, f.err }
func (f *fakeMgr) Search(context.Context, string) ([]manager.Package, error)   { return f.pkgs, f.err }
func (f *fakeMgr) Info(context.Context, string) (manager.Package, error)       { return manager.Package{}, f.err }
func (f *fakeMgr) Install(context.Context, ...string) error                    { return f.err }
func (f *fakeMgr) Uninstall(context.Context, ...string) error                  { return f.err }
func (f *fakeMgr) Update(context.Context, ...string) error                     { return f.err }

func TestSearchFansOutAndMergesResults(t *testing.T) {
	mgrs := []manager.Manager{
		&fakeMgr{id: "brew", pkgs: []manager.Package{{Name: "ripgrep", Manager: "brew", Version: "14.1.0"}}},
		&fakeMgr{id: "cargo", pkgs: []manager.Package{{Name: "ripgrep", Manager: "cargo", Version: "14.0.0"}}},
	}
	o := New(Ranking{Preferred: []string{"brew", "cargo"}})
	res, errs := o.Search(context.Background(), mgrs, "ripgrep")
	if len(errs) != 0 {
		t.Fatalf("errors: %v", errs)
	}
	if len(res) != 2 {
		t.Fatalf("len = %d", len(res))
	}
	// brew should rank first (preferred order).
	sort.Slice(res, func(i, j int) bool { return res[i].Rank < res[j].Rank })
	if res[0].Pkg.Manager != "brew" {
		t.Fatalf("first = %s", res[0].Pkg.Manager)
	}
}

func TestSearchCollectsPartialErrors(t *testing.T) {
	mgrs := []manager.Manager{
		&fakeMgr{id: "ok", pkgs: []manager.Package{{Name: "x", Manager: "ok"}}},
		&fakeMgr{id: "boom", err: errors.New("kaboom")},
	}
	o := New(Ranking{})
	res, errs := o.Search(context.Background(), mgrs, "x")
	if len(res) != 1 {
		t.Fatalf("len res = %d", len(res))
	}
	if len(errs) != 1 {
		t.Fatalf("len errs = %d", len(errs))
	}
}
