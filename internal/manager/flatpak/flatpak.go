// Package flatpak is the Flatpak adapter (per-user install scope).
package flatpak

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "flatpak"} }

func (a *Adapter) ID() string                { return "flatpak" }
func (a *Adapter) DisplayName() string       { return "Flatpak" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

func tsv(b []byte) [][]string {
	var rows [][]string
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := strings.TrimRight(s.Text(), "\r")
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, "\t"))
	}
	return rows
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", "--columns=name,application,version,description", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range tsv(res.Stdout) {
		if len(row) < 2 {
			continue
		}
		p := manager.Package{Name: row[0], Manager: a.ID(), Extra: map[string]string{"app_id": row[1]}}
		if len(row) > 2 {
			p.Version = row[2]
		}
		if len(row) > 3 {
			p.Description = row[3]
		}
		out = append(out, p)
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--app", "--columns=name,application,version"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range tsv(res.Stdout) {
		if len(row) < 2 {
			continue
		}
		p := manager.Package{Name: row[0], Manager: a.ID(), Installed: true, Extra: map[string]string{"app_id": row[1]}}
		if len(row) > 2 {
			p.Version = row[2]
		}
		out = append(out, p)
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"remote-ls", "--updates", "--columns=name,application"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range tsv(res.Stdout) {
		if len(row) < 1 {
			continue
		}
		out = append(out, manager.Package{Name: row[0], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", "--show-summary", "--show-permissions", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	return manager.Package{Name: name, Manager: a.ID(), Description: string(res.Stdout)}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install", "-y", "--user"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"uninstall", "-y"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"update", "-y"}
	if len(names) > 0 {
		args = append(args, names...)
	}
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
