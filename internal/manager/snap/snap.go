// Package snap is the Ubuntu/Linux Snap adapter.
package snap

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "snap"} }

func (a *Adapter) ID() string          { return "snap" }
func (a *Adapter) DisplayName() string { return "Snap" }
func (a *Adapter) OSes() []manager.OS  { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	return op == manager.OpInstall || op == manager.OpUninstall || op == manager.OpUpdate
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"find", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	return parseSnapTable(res.Stdout, a.ID(), false, true), nil
}
func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	return parseSnapTable(res.Stdout, a.ID(), true, false), nil
}
func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"refresh", "--list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	return parseSnapTable(res.Stdout, a.ID(), true, false), nil
}

func parseSnapTable(b []byte, pmID string, installed bool, withSummary bool) []manager.Package {
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(b))
	first := true
	for s.Scan() {
		line := s.Text()
		if first && strings.HasPrefix(line, "Name") {
			first = false
			continue
		}
		first = false
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		p := manager.Package{Name: f[0], Version: f[1], Manager: pmID, Installed: installed}
		if withSummary && len(f) > 4 {
			p.Description = strings.Join(f[4:], " ")
		}
		out = append(out, p)
	}
	return out
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	p := manager.Package{Name: name, Manager: a.ID()}
	sc := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "summary:"):
			p.Description = strings.TrimSpace(strings.TrimPrefix(line, "summary:"))
		case strings.HasPrefix(line, "publisher:"):
			p.Extra = map[string]string{"publisher": strings.TrimSpace(strings.TrimPrefix(line, "publisher:"))}
		}
	}
	return p, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"remove"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"refresh"}
	if len(names) > 0 {
		args = append(args, names...)
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
