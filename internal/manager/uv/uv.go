// Package uv wraps the uv tool subcommand.
package uv

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "uv"} }

func (a *Adapter) ID() string                { return "uv" }
func (a *Adapter) DisplayName() string       { return "uv tool" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// uv has no built-in registry search; uv installs from PyPI. Defer search
	// to PyPI exact-match (same pattern as pip adapter, but here we keep it simple).
	return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNotFound,
		Err: errString("uv: search not implemented; use pip adapter for PyPI search")}
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"tool", "list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) < 2 {
			continue
		}
		ver := strings.TrimPrefix(f[1], "v")
		out = append(out, manager.Package{Name: f[0], Version: ver, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"tool", "install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"tool", "uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		return a.run(ctx, manager.OpUpdate, []string{"tool", "upgrade", "--all"})
	}
	return a.run(ctx, manager.OpUpdate, append([]string{"tool", "upgrade"}, names...))
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}

type errString string

func (e errString) Error() string { return string(e) }
