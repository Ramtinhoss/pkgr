// Package _template is a copy-paste starting point for new adapters.
// Replace _template with the adapter ID throughout.
package _template

import (
	"context"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string // override for tests
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "_template"} }

func (a *Adapter) ID() string                  { return "_template" }
func (a *Adapter) DisplayName() string         { return "_template" }
func (a *Adapter) OSes() []manager.OS          { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope        { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool   { return false }
func (a *Adapter) Detect() bool                { return false }

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error)            { return nil, nil }
func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error)        { return nil, nil }
func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error)  { return manager.Package{}, nil }
func (a *Adapter) Install(ctx context.Context, names ...string) error              { return nil }
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error            { return nil }
func (a *Adapter) Update(ctx context.Context, names ...string) error               { return nil }
