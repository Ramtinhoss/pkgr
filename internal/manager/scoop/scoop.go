// Package scoop is the Windows Scoop adapter.
package scoop

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "scoop"} }

func (a *Adapter) ID() string                { return "scoop" }
func (a *Adapter) DisplayName() string       { return "Scoop" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q, "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var entries []struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Bucket      string `json:"bucket"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range entries {
		out = append(out, manager.Package{
			Name: e.Name, Version: e.Version, Description: e.Description, Manager: a.ID(),
			Extra: map[string]string{"bucket": e.Bucket},
		})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var entries []struct {
		Name    string `json:"Name"`
		Version string `json:"Version"`
		Source  string `json:"Source"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range entries {
		out = append(out, manager.Package{Name: e.Name, Version: e.Version, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"status", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var entries []struct {
		Name             string `json:"Name"`
		InstalledVersion string `json:"Installed Version"`
		LatestVersion    string `json:"Latest Version"`
	}
	if err := json.Unmarshal(res.Stdout, &entries); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range entries {
		out = append(out, manager.Package{Name: e.Name, Version: e.InstalledVersion, Latest: e.LatestVersion, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", name, "--json"}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	var v struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Homepage    string `json:"homepage"`
	}
	if err := json.Unmarshal(res.Stdout, &v); err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeParseError, Err: err}
	}
	return manager.Package{Name: v.Name, Version: v.Version, Description: v.Description, Homepage: v.Homepage, Manager: a.ID()}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update"}
	if len(names) > 0 {
		args = append(args, names...)
	} else {
		args = append(args, "*")
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
