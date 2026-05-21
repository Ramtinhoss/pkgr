// Package nix wraps Nix's modern CLI (nix profile).
package nix

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "nix"} }

func (a *Adapter) ID() string                { return "nix" }
func (a *Adapter) DisplayName() string       { return "Nix" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", "nixpkgs", q, "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	body := map[string]struct {
		Pname       string `json:"pname"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}{}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for attr, v := range body {
		out = append(out, manager.Package{
			Name: v.Pname, Version: v.Version, Description: v.Description,
			Manager: a.ID(), Extra: map[string]string{"attr": attr},
		})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"profile", "list", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var body struct {
		Elements []struct {
			AttrPath   string   `json:"attrPath"`
			StorePaths []string `json:"storePaths"`
		} `json:"elements"`
	}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for _, e := range body.Elements {
		name := e.AttrPath
		if idx := strings.LastIndex(name, "."); idx >= 0 {
			name = name[idx+1:]
		}
		out = append(out, manager.Package{Name: name, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	// nix profile upgrade --dry-run prints upgrade candidates; parse line-by-line.
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"profile", "upgrade", "--dry-run", ".*"}})
	var out []manager.Package
	for _, line := range strings.Split(string(res.Stdout), "\n") {
		if strings.Contains(line, "would be replaced by") {
			out = append(out, manager.Package{Name: line, Manager: a.ID(), Installed: true})
		}
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	pkgs, err := a.Search(ctx, name)
	if err != nil || len(pkgs) == 0 {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound}
	}
	return pkgs[0], nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	args := []string{"profile", "install"}
	for _, n := range names {
		args = append(args, "nixpkgs#"+n)
	}
	return a.run(ctx, manager.OpInstall, args)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"profile", "remove"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	if len(names) == 0 {
		return a.run(ctx, manager.OpUpdate, []string{"profile", "upgrade", ".*"})
	}
	return a.run(ctx, manager.OpUpdate, append([]string{"profile", "upgrade"}, names...))
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
