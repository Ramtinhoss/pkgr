// Package cargo wraps cargo install for binary crates.
package cargo

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "cargo"} }

func (a *Adapter) ID() string                { return "cargo" }
func (a *Adapter) DisplayName() string       { return "cargo" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux, manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

var searchRe = regexp.MustCompile(`^([\w\-_]+)\s*=\s*"([^"]+)"\s*(?:#\s*(.*))?$`)

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q, "--limit", "25"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		m := searchRe.FindStringSubmatch(s.Text())
		if len(m) >= 3 {
			p := manager.Package{Name: m[1], Version: m[2], Manager: a.ID()}
			if len(m) == 4 {
				p.Description = m[3]
			}
			out = append(out, p)
		}
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"install", "--list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "    ") {
			continue
		}
		line = strings.TrimSuffix(line, ":")
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		ver := strings.TrimPrefix(f[1], "v")
		out = append(out, manager.Package{Name: f[0], Version: ver, Manager: a.ID(), Installed: true})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) { return nil, nil }
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
	// `cargo install <name>` reinstalls; pass --force for explicit update.
	if len(names) == 0 {
		return &manager.Error{Manager: a.ID(), Op: manager.OpUpdate, Code: manager.CodeConflict, Err: errString("cargo update requires explicit names")}
	}
	return a.run(ctx, manager.OpUpdate, append([]string{"install", "--force"}, names...))
}
func (a *Adapter) run(ctx context.Context, op manager.Op, args []string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: args})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err}
	}
	return nil
}

type errString string

func (e errString) Error() string { return string(e) }
