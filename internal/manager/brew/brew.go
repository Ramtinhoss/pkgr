// Package brew is the Homebrew adapter.
package brew

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ramtinhoss/pkgr/internal/manager"
	"github.com/ramtinhoss/pkgr/internal/runner"
)

type Adapter struct {
	Runner *runner.Runner
	Bin    string
}

func New(r *runner.Runner) *Adapter { return &Adapter{Runner: r, Bin: "brew"} }

func (a *Adapter) ID() string                { return "brew" }
func (a *Adapter) DisplayName() string       { return "Homebrew" }
func (a *Adapter) OSes() []manager.OS        { return []manager.OS{manager.Darwin, manager.Linux} }
func (a *Adapter) Scope() manager.Scope      { return manager.ScopeUserGlobal }
func (a *Adapter) NeedsSudo(manager.Op) bool { return false }
func (a *Adapter) Detect() bool {
	_, err := exec.LookPath(a.Bin)
	return err == nil
}

// ---- search ----------------------------------------------------------------

type searchPayload struct {
	Formulae []struct {
		Name     string            `json:"name"`
		Desc     string            `json:"desc"`
		Homepage string            `json:"homepage"`
		Versions map[string]string `json:"versions"`
	} `json:"formulae"`
	Casks []struct {
		Token    string   `json:"token"`
		Desc     string   `json:"desc"`
		Homepage string   `json:"homepage"`
		Version  string   `json:"version"`
		Names    []string `json:"name"`
	} `json:"casks"`
}

func (a *Adapter) Search(ctx context.Context, q string) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{
		Bin:  a.Bin,
		Args: []string{"search", "--formula", "--json=v2", q},
	})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeUnknown, Err: err, Stderr: string(res.Stderr)}
	}
	var p searchPayload
	if err := json.Unmarshal(res.Stdout, &p); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpSearch, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(p.Formulae)+len(p.Casks))
	for _, f := range p.Formulae {
		out = append(out, manager.Package{
			Name:        f.Name,
			Version:     f.Versions["stable"],
			Manager:     a.ID(),
			Description: f.Desc,
			Homepage:    f.Homepage,
		})
	}
	for _, c := range p.Casks {
		name := c.Token
		out = append(out, manager.Package{
			Name:        name,
			Version:     c.Version,
			Manager:     a.ID(),
			Description: c.Desc,
			Homepage:    c.Homepage,
			Extra:       map[string]string{"type": "cask"},
		})
	}
	return out, nil
}

// ---- list ------------------------------------------------------------------

func (a *Adapter) List(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"list", "--versions"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpList, Code: manager.CodeUnknown, Err: err}
	}
	var out []manager.Package
	s := bufio.NewScanner(bytes.NewReader(res.Stdout))
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		ver := ""
		if len(parts) > 1 {
			ver = parts[len(parts)-1]
		}
		out = append(out, manager.Package{
			Name: parts[0], Version: ver, Manager: a.ID(), Installed: true,
		})
	}
	return out, nil
}

// ---- outdated --------------------------------------------------------------

type outdatedPayload struct {
	Formulae []struct {
		Name              string   `json:"name"`
		InstalledVersions []string `json:"installed_versions"`
		CurrentVersion    string   `json:"current_version"`
	} `json:"formulae"`
}

func (a *Adapter) Outdated(ctx context.Context) ([]manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"outdated", "--json=v2"}})
	if err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeUnknown, Err: err}
	}
	var p outdatedPayload
	if err := json.Unmarshal(res.Stdout, &p); err != nil {
		return nil, &manager.Error{Manager: a.ID(), Op: manager.OpOutdated, Code: manager.CodeParseError, Err: err}
	}
	out := make([]manager.Package, 0, len(p.Formulae))
	for _, f := range p.Formulae {
		ver := ""
		if len(f.InstalledVersions) > 0 {
			ver = f.InstalledVersions[0]
		}
		out = append(out, manager.Package{
			Name: f.Name, Version: ver, Latest: f.CurrentVersion, Manager: a.ID(), Installed: true,
		})
	}
	return out, nil
}

// ---- info -------------------------------------------------------------------

func (a *Adapter) Info(ctx context.Context, name string) (manager.Package, error) {
	res, err := a.Runner.Run(ctx, runner.Cmd{Bin: a.Bin, Args: []string{"info", "--json=v2", name}})
	if err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeUnknown, Err: err}
	}
	var arr []struct {
		Name      string            `json:"name"`
		Desc      string            `json:"desc"`
		Homepage  string            `json:"homepage"`
		Versions  map[string]string `json:"versions"`
		Installed []struct {
			Version string `json:"version"`
		} `json:"installed"`
	}
	if err := json.Unmarshal(res.Stdout, &arr); err != nil {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeParseError, Err: err}
	}
	if len(arr) == 0 {
		return manager.Package{}, &manager.Error{Manager: a.ID(), Op: manager.OpInfo, Code: manager.CodeNotFound, Err: fmt.Errorf("brew: not found: %s", name)}
	}
	f := arr[0]
	p := manager.Package{
		Name:        f.Name,
		Version:     f.Versions["stable"],
		Manager:     a.ID(),
		Description: f.Desc,
		Homepage:    f.Homepage,
	}
	if len(f.Installed) > 0 {
		p.Installed = true
		p.Version = f.Installed[0].Version
		p.Latest = f.Versions["stable"]
	}
	return p, nil
}

// ---- mutations -------------------------------------------------------------

func (a *Adapter) Install(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpInstall, append([]string{"install"}, names...)...)
}
func (a *Adapter) Uninstall(ctx context.Context, names ...string) error {
	return a.exec(ctx, manager.OpUninstall, append([]string{"uninstall"}, names...)...)
}
func (a *Adapter) Update(ctx context.Context, names ...string) error {
	args := []string{"upgrade"}
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
