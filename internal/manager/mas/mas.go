// Package mas wraps the Mac App Store CLI (mas).
package mas

import (
	"bufio"
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "mas"} }

func (a *Adapter) ID() string                { return "mas" }
func (a *Adapter) DisplayName() string       { return "Mac App Store" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

var rowRe = regexp.MustCompile(`^\s*(\d+)\s+(.+?)\s+\((.+)\)\s*$`)

func parseMas(b []byte, pmID string, installed bool, outdated bool) []manager.Package {
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		m := rowRe.FindStringSubmatch(s.Text())
		if len(m) != 4 {
			continue
		}
		p := manager.Package{Name: strings.TrimSpace(m[2]), Manager: pmID, Installed: installed, Extra: map[string]string{"appid": m[1]}}
		if outdated {
			parts := strings.Split(m[3], " -> ")
			if len(parts) == 2 {
				p.Version = parts[0]
				p.Latest = parts[1]
			}
		} else {
			p.Version = m[3]
		}
		out = append(out, p)
	}
	return out
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	return parseMas(res.Stdout, a.ID(), false, false), nil
}
func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	return parseMas(res.Stdout, a.ID(), true, false), nil
}
func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	return parseMas(res.Stdout, a.ID(), true, true), nil
}
func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	return manager.Package{Name: name, Manager: a.ID()}, nil
}
func (a *Adapter) Install(ctx context.Context, names ...string) error {
	// mas install requires app IDs, not names. Caller must pass numeric IDs.
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
