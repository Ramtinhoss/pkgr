// Package mise wraps the mise (formerly rtx) version manager.
package mise

import (
	"bufio"
	"bytes"
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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "mise"} }

func (a *Adapter) ID() string                { return "mise" }
func (a *Adapter) DisplayName() string       { return "mise" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"plugins", "ls-remote"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		name := strings.TrimSpace(s.Text())
		if name == "" {
			continue
		}
		if q != "" && !strings.Contains(name, q) {
			continue
		}
		out = append(out, manager.Package{Name: name, Manager: a.ID(), Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"ls", "--json"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	body := map[string][]struct {
		Version string `json:"version"`
	}{}
	if err := json.Unmarshal(res.Stdout, &body); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeParseError, Err: err}
	}
	var out []manager.Package
	for plugin, vers := range body {
		for _, v := range vers {
			out = append(out, manager.Package{Name: plugin, Version: v.Version, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
		}
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "--json"}})
	if err != nil {
		return nil, nil
	}
	body := []struct {
		Plugin    string `json:"plugin"`
		Requested string `json:"requested"`
		Latest    string `json:"latest"`
	}{}
	_ = json.Unmarshal(res.Stdout, &body)
	out := make([]manager.Package, 0, len(body))
	for _, e := range body {
		out = append(out, manager.Package{Name: e.Plugin, Version: e.Requested, Latest: e.Latest, Manager: a.ID(), Installed: true, Extra: map[string]string{"kind": "toolchain"}})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade"}
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
