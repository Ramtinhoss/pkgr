// Package dnf is the Fedora/RHEL DNF adapter.
package dnf

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "dnf"} }

func (a *Adapter) ID() string          { return "dnf" }
func (a *Adapter) DisplayName() string { return "DNF" }
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
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "===") || strings.HasPrefix(line, "Last metadata") || line == "" {
			continue
		}
		// format: name.arch : description
		idx := strings.Index(line, " : ")
		if idx < 0 {
			continue
		}
		nameArch := line[:idx]
		name := nameArch
		if dot := strings.LastIndex(nameArch, "."); dot > 0 {
			name = nameArch[:dot]
		}
		out = append(out, manager.Package{Name: name, Description: line[idx+3:], Manager: a.ID()})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--installed"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	return parseDnfList(res.Stdout, a.ID(), false), nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, _ := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"check-update"}})
	// exit code 100 = updates available; treat that as success
	return parseDnfList(res.Stdout, a.ID(), true), nil
}

func parseDnfList(b []byte, pmID string, outdated bool) []manager.Package {
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()
		if line == "" || strings.HasPrefix(line, "Installed") || strings.HasPrefix(line, "Last") {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		nameArch := f[0]
		name := nameArch
		if dot := strings.LastIndex(nameArch, "."); dot > 0 {
			name = nameArch[:dot]
		}
		p := manager.Package{Name: name, Version: f[1], Manager: pmID, Installed: !outdated}
		if outdated {
			p.Latest = f[1]
			p.Version = ""
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
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := s.Text()
		switch {
		case strings.HasPrefix(line, "Version "):
			p.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version :"))
		case strings.HasPrefix(line, "URL "):
			p.Homepage = strings.TrimSpace(strings.TrimPrefix(line, "URL :"))
		case strings.HasPrefix(line, "Summary "):
			p.Description = strings.TrimSpace(strings.TrimPrefix(line, "Summary :"))
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
