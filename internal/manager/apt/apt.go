// Package apt is the Debian/Ubuntu APT adapter.
package apt

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string // "apt" by default; some ops use "apt-cache"
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "apt"} }

func (a *Adapter) ID() string          { return "apt" }
func (a *Adapter) DisplayName() string { return "APT" }
func (a *Adapter) OSes() []manager.OS  { return []manager.OS{manager.Linux} }
func (a *Adapter) Scope() manager.Scope { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(op manager.Op) bool {
	switch op {
	case manager.OpInstall, manager.OpUninstall, manager.OpUpdate:
		return true
	}
	return false
}
func (a *Adapter) Detect() bool { _, err := exec.LookPath(a.Bin); return err == nil }

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: "apt-cache", Args: []string{"search", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		idx := strings.Index(line, " - ")
		if idx < 0 {
			continue
		}
		out = append(out, manager.Package{
			Name: line[:idx], Description: line[idx+3:], Manager: a.ID(),
		})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--installed"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	return parseAptList(res.Stdout, a.ID(), false), nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--upgradable"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	return parseAptList(res.Stdout, a.ID(), true), nil
}

func parseAptList(b []byte, pmID string, upgradable bool) []manager.Package {
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "Listing") {
			continue
		}
		// format: <name>/<dist>,now <ver> <arch> [installed,...]
		slash := strings.Index(line, "/")
		if slash < 0 {
			continue
		}
		name := line[:slash]
		rest := strings.Fields(line[slash:])
		if len(rest) < 2 {
			continue
		}
		ver := rest[1]
		p := manager.Package{Name: name, Version: ver, Manager: pmID, Installed: !upgradable}
		if upgradable {
			// rest[len-1] like "[upgradable from: X]"; salvage X if present
			if idx := strings.Index(line, "from: "); idx > 0 {
				p.Version = strings.TrimSuffix(line[idx+6:], "]")
				p.Latest = ver
			}
		}
		out = append(out, p)
	}
	return out
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: "apt-cache", Args: []string{"show", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	p := manager.Package{Name: name, Manager: a.ID()}
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "Version: "):
			p.Version = strings.TrimPrefix(line, "Version: ")
		case strings.HasPrefix(line, "Homepage: "):
			p.Homepage = strings.TrimPrefix(line, "Homepage: ")
		case strings.HasPrefix(line, "Description: "):
			p.Description = strings.TrimPrefix(line, "Description: ")
		}
	}
	return p, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install", "-y"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"remove", "-y"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade", "-y"}
	if len(names) > 0 {
		args = append([]string{"install", "-y", "--only-upgrade"}, names...)
	}
	return a.exec(ctx, manager.OpUpdate, args...)
}
func (a *Adapter) exec(ctx context.Context, op manager.Op, args ...string) error {
	_, err := a.Runner.Run(ctx, runner.Cmd{Bin: "apt-get", Args: args, Sudo: a.NeedsSudo(op)})
	if err != nil {
		return &manager.Error{Manager: a.ID(), Op: op, Code: manager.CodeUnknown, Err: err, Cmd: fmt.Sprintf("apt-get %s", strings.Join(args, " "))}
	}
	return nil
}
