// Package asdf wraps the asdf version manager.
package asdf

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "asdf"} }

func (a *Adapter) ID() string                { return "asdf" }
func (a *Adapter) DisplayName() string       { return "asdf" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"plugin", "list", "all"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		f := strings.Fields(s.Text())
		if len(f) == 0 {
			continue
		}
		if q != "" && !strings.Contains(f[0], q) {
			continue
		}
		p := manager.Package{Name: f[0], Manager: a.ID(), Extra: map[string]string{"kind": "toolchain"}}
		if len(f) > 1 {
			p.Homepage = f[1]
		}
		out = append(out, p)
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	var current string
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			current = strings.TrimSpace(line)
			continue
		}
		ver := strings.TrimSpace(strings.TrimPrefix(line, "  *"))
		out = append(out, manager.Package{Name: current, Version: ver, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	installed, err := a.List(ctx)
	if err != nil {
		return nil, err
	}
	var out []manager.Package
	for _, p := range installed {
		res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"latest", p.Name}})
		latest := strings.TrimSpace(string(res.Stdout))
		if latest != "" && latest != p.Version {
			p.Latest = latest
			out = append(out, p)
		}
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	// asdf install <plugin> <version>; treat each names[i] as "plugin@version"
	for _, n := range names {
		parts := strings.SplitN(n, "@", 2)
		args := []string{"install", parts[0]}
		if len(parts) == 2 {
			args = append(args, parts[1])
		} else {
			args = append(args, "latest")
		}
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpInstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}

func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	for _, n := range names {
		parts := strings.SplitN(n, "@", 2)
		if len(parts) != 2 {
			return &manager.Error{Manager: a.ID(), Op: manager.OpUninstall, Code: manager.CodeConflict, Err: errString("asdf uninstall needs plugin@version")}
		}
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"uninstall", parts[0], parts[1]}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpUninstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}

func (a *Adapter) Update(ctx context.Context, names ...string) error {
	// asdf update doesn't update plugins/versions; treat as plugin-update.
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"plugin", "update", "--all"}})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}

type errString string

func (e errString) Error() string { return string(e) }
