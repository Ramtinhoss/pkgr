// Package rustup wraps the rustup toolchain manager.
package rustup

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "rustup"} }

func (a *Adapter) ID() string                { return "rustup" }
func (a *Adapter) DisplayName() string       { return "rustup" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// rustup toolchains are well-known constants; expose the common ones.
	known := []string{"stable", "beta", "nightly"}
	var out []manager.Package
	for _, n := range known {
		if q != "" && !strings.Contains(n, q) {
			continue
		}
		out = append(out, manager.Package{Name: n, Manager: a.ID(), Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"toolchain", "list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		name := strings.Fields(line)[0]
		out = append(out, manager.Package{Name: name, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"check"}})
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		if !strings.Contains(line, "Update available") {
			continue
		}
		// "<toolchain> - Update available : X -> Y"
		dash := strings.Index(line, " - ")
		if dash < 0 {
			continue
		}
		name := strings.Fields(line[:dash])[0]
		idx := strings.LastIndex(line, " -> ")
		if idx < 0 {
			continue
		}
		latest := strings.TrimSpace(line[idx+4:])
		out = append(out, manager.Package{Name: name, Latest: latest, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"toolchain", "install", n}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpInstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"toolchain", "uninstall", n}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpUninstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update"}
	if len(names) > 0 {
		args = append(args, names...)
	}
	if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args}); err != nil {
		return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
