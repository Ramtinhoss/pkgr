// Package pnpm wraps pnpm for global packages.
package pnpm

import (
	"context"
	"encoding/json"
	"os/exec"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "pnpm"} }

func (a *Adapter) ID() string                { return "pnpm" }
func (a *Adapter) DisplayName() string       { return "pnpm" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// pnpm has no search command; defer to `npm search --json` if available.
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: "npm", Args: []string{"search", "--json", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var entries []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Links       struct {
			Homepage string `json:"homepage"`
		} `json:"links"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(entries))
	for _, e := range entries {
		out = append(out, manager.Package{Name: e.Name, Version: e.Version, Description: e.Description, Homepage: e.Links.Homepage, Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "-g", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var arr []struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(res.Stdout, &arr); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range arr {
		for name, d := range e.Dependencies {
			out = append(out, manager.Package{Name: name, Version: d.Version, Manager: a.ID(), Installed: true})
		}
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "-g", "--format", "json"}})
	if len(res.Stdout) == 0 {
		return nil, nil
	}
	var m map[string]struct {
		Current string `json:"current"`
		Latest  string `json:"latest"`
	}
	if err := json.Unmarshal(res.Stdout, &m); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for name, v := range m {
		out = append(out, manager.Package{Name: name, Version: v.Current, Latest: v.Latest, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"add", "-g"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"remove", "-g"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update", "-g"}
	if len(names) > 0 {
		args = append(args, names...)
	}
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
