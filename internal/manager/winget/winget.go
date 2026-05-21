// Package winget is the Windows winget adapter. winget prints a fixed-column
// table; we detect column widths from the header row.
package winget

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

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "winget"} }

func (a *Adapter) ID() string                { return "winget" }
func (a *Adapter) DisplayName() string       { return "winget" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Windows} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeSystem }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool              { _, err := exec.LookPath(a.Bin); return err == nil }

type col struct {
	name       string
	start, end int
}

func parseHeader(line string) []col {
	var cols []col
	i := 0
	for i < len(line) {
		for i < len(line) && line[i] == ' ' {
			i++
		}
		start := i
		for i < len(line) && line[i] != ' ' {
			i++
		}
		if start == i {
			break
		}
		cols = append(cols, col{name: line[start:i], start: start, end: i})
	}
	return cols
}

// cell extracts the value for column c from line, honoring the column start
// and stopping at the next column's start.
func cell(line string, cols []col, idx int) string {
	if idx >= len(cols) {
		return ""
	}
	s := cols[idx].start
	if s >= len(line) {
		return ""
	}
	e := len(line)
	if idx+1 < len(cols) {
		e = cols[idx+1].start
	}
	if e > len(line) {
		e = len(line)
	}
	return strings.TrimSpace(line[s:e])
}

func parseTable(b []byte) ([]col, []string) {
	var lines []string
	s := bufio.NewScanner(bytes.NewReader(b))
	for s.Scan() {
		line := s.Text()
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	// find header line (one with "Name" at start)
	headerIdx := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "Name ") || strings.HasPrefix(strings.TrimSpace(l), "Name\t") {
			headerIdx = i
			break
		}
	}
	if headerIdx < 0 {
		return nil, nil
	}
	cols := parseHeader(lines[headerIdx])
	// skip separator
	return cols, lines[headerIdx+2:]
}

func indexOf(cols []col, name string) int {
	for i, c := range cols {
		if c.name == name {
			return i
		}
	}
	return -1
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"search", q}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err}
	}
	cols, body := parseTable(res.Stdout)
	if cols == nil {
		return nil, nil
	}
	idx := indexOf(cols, "Name")
	idIdx := indexOf(cols, "Id")
	verIdx := indexOf(cols, "Version")
	var out []manager.Package
	for _, l := range body {
		name := cell(l, cols, idx)
		if name == "" {
			continue
		}
		out = append(out, manager.Package{
			Name: name, Version: cell(l, cols, verIdx), Manager: a.ID(),
			Extra: map[string]string{"id": cell(l, cols, idIdx)},
		})
	}
	return out, nil
}

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	cols, body := parseTable(res.Stdout)
	if cols == nil {
		return nil, nil
	}
	var out []manager.Package
	for _, l := range body {
		name := cell(l, cols, indexOf(cols, "Name"))
		if name == "" {
			continue
		}
		out = append(out, manager.Package{
			Name:      name,
			Version:   cell(l, cols, indexOf(cols, "Version")),
			Latest:    cell(l, cols, indexOf(cols, "Available")),
			Manager:   a.ID(),
			Installed: true,
		})
	}
	return out, nil
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"upgrade"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	cols, body := parseTable(res.Stdout)
	if cols == nil {
		return nil, nil
	}
	var out []manager.Package
	for _, l := range body {
		name := cell(l, cols, indexOf(cols, "Name"))
		if name == "" {
			continue
		}
		out = append(out, manager.Package{
			Name:      name,
			Version:   cell(l, cols, indexOf(cols, "Version")),
			Latest:    cell(l, cols, indexOf(cols, "Available")),
			Manager:   a.ID(),
			Installed: true,
		})
	}
	return out, nil
}

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"show", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: err}
	}
	return manager.Package{Name: name, Manager: a.ID(), Description: string(res.Stdout)}, nil
}

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpInstall, append([]string{"install"}, names...))
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.run(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...))
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade"}
	if len(names) == 0 {
		args = append(args, "--all")
	} else {
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
