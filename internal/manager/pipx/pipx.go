// Package pipx wraps pipx (isolated Python tools).
package pipx

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "pipx"} }

func (a *Adapter) ID() string                { return "pipx" }
func (a *Adapter) DisplayName() string       { return "pipx" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	// pipx has no search; defer to PyPI exact-match (same approach as pip adapter).
	return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeNotFound,
		Err: errString("pipx: search not implemented; use pip adapter for PyPI search")}
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var body struct {
		Venvs map[string]struct {
			Metadata struct {
				MainPackage struct {
					Name    string `json:"package_or_url"`
					Version string `json:"package_version"`
				} `json:"main_package"`
			} `json:"metadata"`
		} `json:"venvs"`
	}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, v := range body.Venvs {
		out = append(out, manager.Package{Name: v.Metadata.MainPackage.Name, Version: v.Metadata.MainPackage.Version, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"install", n}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpInstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	for _, n := range names {
		if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"uninstall", n}}); err != nil {
			return &manager.Error{Manager: a.ID(), Op: manager.OpUninstall, Code: manager.CodeUnknown, Err: err}
		}
	}
	return nil
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade-all"}
	if len(names) > 0 {
		args = append([]string{"upgrade"}, names...)
	}
	if _, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args}); err != nil {
		return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}

type errString string

func (e errString) Error() string { return string(e) }
