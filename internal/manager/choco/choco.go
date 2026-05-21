// Package choco is the Windows Chocolatey adapter.
// We use `-r` (limited output) which prints pipe-delimited lines.
package choco

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "choco"} }

func (a *Adapter) ID() string          { return "choco" }
func (a *Adapter) DisplayName() string { return "Chocolatey" }
func (a *Adapter) OSes() []manager.OS  { return []manager.OS{manager.Windows} }
func (a *Adapter) Scope() manager.Scope { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	return op == manager.OpInstall || op == manager.OpUninstall || op == manager.OpUpdate
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func parsePipe(b []byte) [][]string {
	var rows [][]string
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()
		if line == "" {
			continue
		}
		rows = append(rows, strings.Split(line, "|"))
	}
	return rows
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q, "-r"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range parsePipe(res.Stdout) {
		if len(row) < 2 {
			continue
		}
		out = append(out, manager.Package{Name: row[0], Version: row[1], Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "-r"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range parsePipe(res.Stdout) {
		if len(row) < 2 {
			continue
		}
		out = append(out, manager.Package{Name: row[0], Version: row[1], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "-r"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	for _, row := range parsePipe(res.Stdout) {
		if len(row) < 3 {
			continue
		}
		out = append(out, manager.Package{Name: row[0], Version: row[1], Latest: row[2], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	return manager.Package{Name: name, Manager: a.ID(), Description: string(res.Stdout)}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install", "-y"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall", "-y"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade", "-y"}
	if len(names) > 0 {
		args = append(args, names...)
	} else {
		args = append(args, "all")
	}
	return a.run(ctx, manager.OpUpdate, args)
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args, Sudo: a.NeedsSudo(op)})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
