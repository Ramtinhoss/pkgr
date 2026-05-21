// Package pacman is the Arch Linux pacman adapter.
package pacman

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "pacman"} }

func (a *Adapter) ID() string          { return "pacman" }
func (a *Adapter) DisplayName() string { return "pacman" }
func (a *Adapter) OSes() []manager.OS  { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	return op == manager.OpInstall || op == manager.OpUninstall || op == manager.OpUpdate
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"-Ss", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	var cur manager.Package
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "    ") {
			cur.Description = strings.TrimSpace(line)
			out = append(out, cur)
			cur = manager.Package{}
			continue
		}
		// repo/name version
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		nameParts := strings.SplitN(parts[0], "/", 2)
		name := nameParts[len(nameParts)-1]
		cur = manager.Package{Name: name, Version: parts[1], Manager: a.ID()}
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"-Q"}})
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
		out = append(out, manager.Package{Name: f[0], Version: f[1], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"-Qu"}})
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		f := strings.Fields(line)
		if len(f) < 4 || f[2] != "->" {
			continue
		}
		out = append(out, manager.Package{Name: f[0], Version: f[1], Latest: f[3], Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"-Si", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	p := manager.Package{Name: name, Manager: a.ID()}
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "Version "):
			p.Version = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(line, "URL "):
			p.Homepage = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		case strings.HasPrefix(line, "Description "):
			p.Description = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}
	return p, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"-S", "--noconfirm"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"-R", "--noconfirm"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"-Syu", "--noconfirm"}
	if len(names) > 0 {
		args = append([]string{"-S", "--noconfirm"}, names...)
	}
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args, Sudo: a.NeedsSudo(op)})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}
